package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"golang.org/x/exp/slices"

	"server-go/common"
	"server-go/database"

	"github.com/patrickmn/go-cache"
	"github.com/uptrace/bun"
)

type UR_RequestData struct {
	DiscordID  Snowflake `json:"userid"`
	Token      string    `json:"token"`
	Comment    string    `json:"comment"`
	ReviewType int       `json:"reviewtype"`
	Sender     struct {
		Username     string `json:"username"`
		ProfilePhoto string `json:"profile_photo"`
		DiscordID    string `json:"discordid"`
	} `json:"sender"`
}

type ReportData struct {
	ReviewID int32  `json:"reviewid"`
	Token    string `json:"token"`
}

type Settings struct {
	bun.BaseModel `bun:"table:users"`

	DiscordID string `bun:"discord_id,type:numeric"`
	Opt       bool   `json:"opt" bun:"opted_out"`
}

func GetReviews(userID int64, offset int) ([]database.UserReview, int, error) {
	return GetReviewsWithOptions(userID, offset, map[string]interface{}{
		"includeReviewsBy": nil,
	})
}

func GetReviewsWithOptions(userID int64, offset int, options map[string]interface{}) ([]database.UserReview, int, error) {
	var reviews []database.UserReview

	count, err := database.DB.NewSelect().
		Model(&reviews).
		Relation("User").
		Where("profile_id = ?", userID).
		Where("\"user\".\"opted_out\" = 'f'").
		OrderExpr("reviewer_id = ? desc, id desc",options["includeReviewsBy"]).Limit(51).
		Offset(offset).
		ScanAndCount(context.Background(), &reviews)

	if err != nil {
		return nil, 0, err
	}

	for i, review := range reviews {
		dbBadges := GetBadgesOfUser(review.User.DiscordID)
		badges := make([]database.UserBadge, len(dbBadges))
		for i, b := range dbBadges {
			badges[i] = database.UserBadge(b)
		}

		if review.User != nil {
			reviews[i].Sender.DiscordID = review.User.DiscordID
			reviews[i].Sender.ProfilePhoto = review.User.AvatarURL
			reviews[i].Sender.Username = review.User.Username
			reviews[i].Sender.ID = review.User.ID
			reviews[i].Sender.Badges = badges
		}
		reviews[i].Timestamp = review.TimestampStr.Unix()
	}

	return reviews, count, nil
}

func GetDBUserViaDiscordID(discordID string) (*database.URUser, error) {
	var user database.URUser
	err := database.DB.NewSelect().Model(&user).Where("discord_id = ?", discordID).Limit(1).Scan(context.Background())

	if err != nil {
		if err.Error() == "sql: no rows in result set" { //SOMEONE TELL ME BETTER WAY TO DO THIS
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func SearchReviews(query string, token string) ([]database.UserReview, error) {
	var reviews []database.UserReview

	user, err := GetDBUserViaToken(token)

	if err != nil {
		return nil, err
	}
	if user.Type != 1 {
		return nil, errors.New("You are not allowed to use this route")
	}

	err = database.DB.NewSelect().Model(&reviews).Relation("User").Where("comment LIKE ?", "%"+query+"%").OrderExpr("ID DESC").Limit(100).Scan(context.Background(), &reviews)
	if err != nil {
		return nil, err
	}

	for i, review := range reviews {
		badges := GetBadgesOfUser(review.User.DiscordID)

		if review.User != nil {
			reviews[i].Sender.DiscordID = review.User.DiscordID
			reviews[i].Sender.ProfilePhoto = review.User.AvatarURL
			reviews[i].Sender.Username = review.User.Username
			reviews[i].Sender.ID = review.User.ID
			reviews[i].Sender.Badges = badges
		}
		reviews[i].Timestamp = review.TimestampStr.Unix()
	}

	return reviews, nil
}

func AddReview(data UR_RequestData) (string, error) {
	var err error
	var reviewer database.URUser

	if data.Token == common.Config.StupidityBotToken {

		user, err := GetDBUserViaDiscordID(data.Sender.DiscordID)
		if err != nil {
			return common.UNAUTHORIZED, err
		}

		if user == nil {
			reviewer, err = CreateUserViaBot(data.Sender.DiscordID, data.Sender.Username, data.Sender.ProfilePhoto)
			if err != nil {
				return common.UNAUTHORIZED, err
			}
		} else {
			reviewer = *user
		}

	} else {
		reviewer, err = GetDBUserViaToken(data.Token)
	}

	if err != nil {
		return "", errors.New(common.INVALID_TOKEN)
	}

	if len(reviewer.Token) == 64 && len(data.Token) != 64 { // try to get rid of old token system as much as possible
		reviewer.Token = data.Token
		database.DB.NewUpdate().Model(reviewer).Set("token = ?", data.Token).Where("id = ?", reviewer.ID).Exec(context.Background())
	}

	if reviewer.OptedOut {
		return "", errors.New(common.OPTED_OUT)
	}

	if !(data.ReviewType == 0 || data.ReviewType == 1) && reviewer.Type != 1 {
		return "", errors.New(common.INVALID_REVIEW_TYPE)
	}

	if reviewer.IsBanned() {
		return "", errors.New("You have been banned from ReviewDB until " + reviewer.BanInfo.BanEndDate.Format("2006-01-02 15:04:05") + "UTC")
	}

	count, _ := GetReviewCountInLastHour(reviewer.ID)
	if count > 20 {
		return "", errors.New("You are reviewing too much")
	}

	if common.LightProfanityDetector.IsProfane(data.Comment) {
		return "", errors.New("Your review contains profanity")
	}

	review := &database.UserReview{

		ProfileID:    int64(data.DiscordID),
		ReviewerID:   reviewer.ID,
		Comment:      data.Comment,
		Type:         int32(data.ReviewType),
		TimestampStr: time.Now(),
	}

	if common.ProfanityDetector.IsProfane(data.Comment) {
		review.ID = -1
		BanUser(reviewer.DiscordID, common.Config.AdminToken, 7, *review)
		SendLoggerWebhook(WebhookData{
			Username: "ReviewDB",
			Content:  "User <@" + reviewer.DiscordID + "> has been banned for 1 week for trying to post a profane review",
			Embeds: []Embed{
				{
					Fields: []EmbedField{
						{
							Name:  "Review Content",
							Value: data.Comment,
						},
						{
							Name:  "ReviewDB ID",
							Value: strconv.Itoa(int(reviewer.ID)),
						},
						{
							Name:  "Reviewed Profile",
							Value: "<@" + strconv.FormatInt(int64(data.DiscordID), 10) + ">",
						},
					},
				},
			},
		})
		return "", errors.New("Because of trying to post a profane review, you have been banned from ReviewDB for 1 week")
	}

	res, err := database.DB.NewUpdate().Where("profile_id = ? AND reviewer_id = ?", data.DiscordID, reviewer.ID).OmitZero().Model(review).Exec(context.Background())
	if err != nil {
		return common.UPDATE_FAILED, err
	}
	//LogAction("UPDATE",review,senderUserID) TODO : DO THIS

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return common.UPDATE_FAILED, err
	}
	if rowsAffected != 0 {
		return common.UPDATED, nil
	}

	_, err = database.DB.NewInsert().Model(review).Exec(context.Background())
	if err != nil {
		return common.ERROR, err
	}
	return common.ADDED, nil
}

func GetIDWithToken(token string) (id int32) {
	database.DB.
		NewSelect().
		Model((*database.URUser)(nil)).
		Column("id").
		Where("token = ? or token = ?", CalculateHash(token), token).
		Scan(context.Background(), &id)
	return
}

func GetDBUserViaToken(token string) (user database.URUser, err error) {
	err = database.DB.
		NewSelect().
		Model(&user).
		Where("token = ? or token = ?", token, CalculateHash(token)).
		Relation("BanInfo").
		Scan(context.Background(), &user)

	if user.BanInfo != nil && user.BanInfo.BanEndDate.Before(time.Now()) {
		user.BanInfo = nil
	}

	return
}

func GetReviewCountInLastHour(userID int32) (int, error) {
	//return 0, nil
	count, err := database.DB.
		NewSelect().Table("reviews").
		Where("reviewer_id = ? AND timestamp > now() - interval '1 hour'", userID).
		Count(context.Background())
	if err != nil {
		return 0, err
	}
	return count, nil
}

func AddUserReviewsUser(code string, clientmod string, authUrl string, ip string) (string, error) {
	//todo make this work exactly same as pyton version
	if authUrl == "" {
		authUrl = "/URauth"
	}
	discordToken, err := ExchangeCode(code, common.Config.Origin+authUrl)
	if err != nil {
		return "", err
	}

	discordUser, err := GetUser(discordToken.AccessToken)
	if err != nil {
		return "", err
	}
	token := GenerateToken()

	user := &database.URUser{
		DiscordID:    discordUser.ID,
		Token:        token,
		Username:     discordUser.Username + "#" + discordUser.Discriminator,
		AvatarURL:    GetProfilePhotoURL(discordUser.ID, discordUser.Avatar),
		Type:         0,
		ClientMods:   []string{clientmod},
		IpHash:       CalculateHash(ip),
		RefreshToken: discordToken.RefreshToken,
	}
	if discordUser.Discriminator == "0" {
		user.Username = discordUser.Username
	}

	dbUser, err := GetDBUserViaDiscordID(discordUser.ID)

	if dbUser != nil {
		if dbUser.Type == -1 {
			return "You have been banned from ReviewDB", errors.New("You have been banned from ReviewDB")
		}

		if !strings.HasPrefix(dbUser.Token, "rdb.") {
			dbUser.Token = token
		}

		if !slices.Contains(dbUser.ClientMods, clientmod) {
			dbUser.ClientMods = append(dbUser.ClientMods, clientmod)
		}

		_, err = database.DB.NewUpdate().Where("id = ?", dbUser.ID).Model(dbUser).Exec(context.Background())
		if err != nil {
			return "", err
		}

		return dbUser.Token, nil
	}

	_, err = database.DB.NewInsert().Model(user).Exec(context.Background())
	if err != nil {
		return "An Error occurred", err
	}

	SendLoggerWebhook(WebhookData{
		Username:  discordUser.Username + "#" + discordUser.Discriminator,
		AvatarURL: GetProfilePhotoURL(discordUser.ID, discordUser.Avatar),
		Content:   fmt.Sprintf("User <@%s> has been registered to ReviewDB from %s", discordUser.ID, clientmod),
	})

	return token, nil
}

func GetReview(id int32) (rep database.UserReview, err error) {
	rep = database.UserReview{}
	err = database.DB.NewSelect().Model(&rep).Relation("User").Where("user_review.id = ?", id).Scan(context.Background(), &rep)
	if err != nil {
		return rep, err
	}

	badges := GetBadgesOfUser(rep.User.DiscordID)

	if rep.User != nil {
		rep.Sender = database.Sender{}
		rep.Sender.DiscordID = rep.User.DiscordID
		rep.Sender.ProfilePhoto = rep.User.AvatarURL
		rep.Sender.Username = rep.User.Username
		rep.Sender.ID = rep.User.ID
		rep.Sender.Badges = badges
	}
	rep.Timestamp = rep.TimestampStr.Unix()

	return
}

func ReportReview(reviewID int32, token string) error {

	user, err := GetDBUserViaToken(token)
	if err != nil {
		return errors.New(common.INVALID_REVIEW)
	}

	if user.IsBanned() {
		return errors.New("You cant report reviews while banned")
	}

	reportCount, _ := GetReportCountInLastHour(user.ID)

	if reportCount > 20 {
		return errors.New("You are reporting too much")
	}

	count, _ := database.DB.NewSelect().Model(&database.ReviewReport{}).Where("review_id = ? AND reporter_id = ?", reviewID, user.ID).Count(context.Background())
	if count > 0 {
		return errors.New("You have already reported this review")
	}

	review, err := GetReview(reviewID)
	if err != nil {
		return errors.New("Invalid Review ID")
	}

	if review.Sender.DiscordID == user.DiscordID {
		return errors.New("You cant report your own reviews")
	}

	reportedUser, _ := GetDBUserViaID(review.ReviewerID)

	report := database.ReviewReport{
		ReviewID:   reviewID,
		ReporterID: user.ID,
	}

	reviewedUsername := "?"
	if reviewedUser, err := ArikawaState.User(discord.UserID(review.ProfileID)); err == nil {
		reviewedUsername = reviewedUser.Tag()
	}

	webhookData := WebhookData{
		Username: "ReviewDB",
		Content:  "Reported Review",
		Components: []WebhookComponent{
			{
				Type: 1,
				Components: []WebhookComponent{
					{
						Type:     2,
						Label:    "Delete Review",
						Style:    4,
						CustomID: fmt.Sprintf("delete_review:%d", reviewID),
						Emoji: WebhookEmoji{
							Name: "🗑️",
						},
					},
					{
						Type:     2,
						Label:    "Ban User",
						Style:    4,
						CustomID: fmt.Sprintf("ban_select:%s:%d", reportedUser.DiscordID, reviewID), //string(reportedUser.DiscordID)
						Emoji: WebhookEmoji{
							Name:     "banned",
							ID:       "590237837299941382",
							Animated: true,
						},
					},
					{
						Type:     2,
						Label:    "Delete Review and Ban User",
						Style:    4,
						CustomID: fmt.Sprintf("select_delete_and_ban:%d:%s", reviewID, string(reportedUser.DiscordID)),
						Emoji: WebhookEmoji{
							Name:     "banned",
							ID:       "590237837299941382",
							Animated: true,
						},
					},
				},
			},
		},
		Embeds: []Embed{
			{
				Fields: []EmbedField{
					{
						Name:  "**Review ID**",
						Value: fmt.Sprint(review.ID),
					},
					{
						Name:  "**Content**",
						Value: fmt.Sprint(review.Comment),
					},
					{
						Name:  "**Author**",
						Value: formatUser(reportedUser.Username, reportedUser.ID, reportedUser.DiscordID),
					},
					{
						Name:  "**Reviewed User**",
						Value: formatUser(reviewedUsername, 0, strconv.FormatInt(review.ProfileID, 10)),
					},
					{
						Name:  "**Reporter**",
						Value: formatUser(user.Username, user.ID, user.DiscordID),
					},
				},
			},
		},
	}

	if reportedUser.DiscordID != user.DiscordID {
		webhookData.Components[0].Components = append(webhookData.Components[0].Components, WebhookComponent{
			Type:     2,
			Label:    "Ban Reporter",
			Style:    4,
			CustomID: fmt.Sprintf("ban_select:" + user.DiscordID + ":" + "0"),
			Emoji: WebhookEmoji{
				Name:     "banned",
				ID:       "590237837299941382",
				Animated: true,
			},
		})
	}

	err = SendReportWebhook(webhookData)

	if err != nil {
		println(err.Error())
	}

	database.DB.NewInsert().Model(&report).Exec(context.Background())
	return nil
}

func formatUser(username string, id int32, discordId string) string {
	if id == 0 {
		return fmt.Sprintf("Username: %v\nDiscord ID: %v (<@%v>)", username, discordId, discordId)
	}
	return fmt.Sprintf("Username: %v\nDiscord ID: %v (<@%v>)\nReviewDB ID: %v", username, discordId, discordId, id)
}

func GetReports() (reports []database.ReviewReport, err error) {
	reports = []database.ReviewReport{}
	err = database.DB.NewSelect().Model(&reports).Scan(context.Background(), &reports)
	return
}

func IsUserAdminDC(discordid int64) bool {
	count, _ := database.DB.NewSelect().Model(&database.URUser{}).Where("discord_id = ? and type = 1", discordid).Count(context.Background())

	if count > 0 {
		return true
	}
	return false
}

func GetDBUserViaID(id int32) (user database.URUser, err error) {
	user = database.URUser{}
	err = database.DB.NewSelect().Model(&user).Where("ur_user.id = ?", id).Relation("BanInfo").Scan(context.Background(), &user)
	if user.BanInfo != nil && user.BanInfo.BanEndDate.Before(time.Now()) {
		user.BanInfo = nil
	}
	return
}

func DeleteReview(reviewID int32, token string) (err error) {
	review, err := GetReview(reviewID)
	if err != nil {
		fmt.Println(err.Error())
		return errors.New("Invalid Review ID")
	}
	user, err := GetDBUserViaToken(token)

	if (review.User.DiscordID == user.DiscordID) || user.IsAdmin() || token == common.Config.AdminToken {
		LogAction("DELETE", review, user.ID)

		_, err = database.DB.NewDelete().Model(&review).Where("id = ?", reviewID).Exec(context.Background())
		return nil
	}
	return errors.New("You are not allowed to delete this review")
}

func GetBadgesOfUser(discordid string) []database.UserBadge {
	userBadges := []database.UserBadge{}

	badges, _ := GetAllBadges()
	for _, badge := range badges {

		if badge.TargetDiscordID == discordid {
			userBadges = append(userBadges, badge)
		}
	}
	return userBadges
}

func GetAllBadges() (badges []database.UserBadge, err error) {

	cachedBadges, found := common.Cache.Get("badges")
	if found {
		badges = cachedBadges.([]database.UserBadge)
		return
	}

	badges = []database.UserBadge{}
	err = database.DB.NewSelect().Model(&badges).Scan(context.Background(), &badges)

	users := []database.URUser{}

	database.DB.NewSelect().Distinct().Model(&users).Column("discord_id", "type").Where("type = ? or type = ?", 1, -1).Scan(context.Background(), &users)

	for _, user := range users {
		if user.Type == 1 {
			badges = append(badges, database.UserBadge{
				TargetDiscordID: user.DiscordID,
				Name:            "Admin",
				Icon:            "https://cdn.discordapp.com/emojis/1040004306100826122.gif?size=128",
				RedirectURL:     "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
				Description:     "This user is an admin of ReviewDB.",
			})
		} else {
			badges = append(badges, database.UserBadge{
				TargetDiscordID: user.DiscordID,
				Name:            "Banned",
				Icon:            "https://cdn.discordapp.com/emojis/399233923898540053.gif?size=128",
				RedirectURL:     "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
				Description:     "This user is banned from ReviewDB.",
			})
		}

	}

	common.Cache.Set("badges", badges, cache.DefaultExpiration)
	return
}

func GetVencordBadges() error {
	//todo eta:never
	return nil
}

func GetURUserCount() (count int, err error) {
	return database.DB.NewSelect().Model(&database.URUser{}).Count(context.Background())
}

func GetReviewCount() (count int, err error) {
	return database.DB.NewSelect().Model(&database.UserReview{}).Count(context.Background())
}

func GetLastReviewID(userID string) int32 {
	review := database.UserReview{}

	err := database.DB.NewSelect().Model(&database.UserReview{}).Where("profile_id = ?", userID).Order("id DESC").Limit(1).Scan(context.Background(), &review)

	if err != nil {
		return 0
	}

	return review.ID
}

func BanUser(userToBan string, adminToken string, banDuration int32, review database.UserReview) error {
	user := database.URUser{}

	admin, _ := GetDBUserViaToken(adminToken)
	if adminToken != common.Config.AdminToken && !admin.IsAdmin() {
		return errors.New("You are not allowed to ban users")
	}

	/*
		AdminDiscordID := 0
		if token != common.Config.AdminToken {
			admin , _ := GetDBUserViaToken(token)
			AdminDiscordID = admin.DiscordID
		}
	*/

	database.DB.NewSelect().Model(&user).Where("discord_id = ?", userToBan).Scan(context.Background(), &user)

	if user.Type == 1 {
		return errors.New("You can't ban an admin")
	}
	if user.WarningCount >= 3 {
		_, err := database.DB.NewUpdate().Model(&database.URUser{}).Where("discord_id = ?", userToBan).Set("type = -1").Exec(context.Background())
		if err != nil {
			return err
		}
		return nil
	}

	var banData database.ReviewDBBanLog

	if review.ID != 0 {
		banData = database.ReviewDBBanLog{
			DiscordID:       userToBan,
			ReviewID:        review.ID,
			BanEndDate:      time.Now().AddDate(0, 0, int(banDuration)),
			ReviewContent:   review.Comment,
			ReviewTimestamp: review.TimestampStr,
		}

	} else {
		banData = database.ReviewDBBanLog{
			DiscordID:  userToBan,
			BanEndDate: time.Now().AddDate(0, 0, int(banDuration)),
		}
	}

	_, err := database.DB.NewInsert().Model(&banData).Exec(context.Background())
	if err != nil {
		return err
	}

	_, err = database.DB.NewUpdate().Model(&database.URUser{}).Where("discord_id = ?", userToBan).Set("ban_id = ?", banData.ID).Set("warning_count = warning_count + 1").Exec(context.Background())

	if err != nil {
		return err
	}
	return nil
}

func GetAdmins() (users []string, err error) {
	users = []string{}
	userlist := []database.AdminUser{}

	err = database.DB.NewSelect().Distinct().Model(&database.AdminUser{}).Where("type = 1").Scan(context.Background(), &userlist)

	for _, user := range userlist {
		users = append(users, user.DiscordID)
	}
	return
}

func LogAction(action string, review database.UserReview, userid int32) {
	log := database.ActionLog{}

	log.UserID = review.ProfileID
	log.Action = action
	log.ReviewID = review.ID
	log.SenderUserID = review.ReviewerID
	log.Comment = review.Comment
	log.ActionUserID = userid

	_, err := database.DB.NewInsert().Model(&log).Exec(context.Background())
	if err != nil {
		fmt.Println(err)
	}
}

func CreateUserViaBot(discordid string, username string, profilePhoto string) (database.URUser, error) {
	user := database.URUser{}

	user.DiscordID = discordid
	user.Username = username
	user.Type = 0
	user.WarningCount = 0
	user.ClientMods = []string{"discordbot"}
	user.AvatarURL = profilePhoto
	user.Token = discordid

	_, err := database.DB.NewInsert().Model(&user).Exec(context.Background())
	if err != nil {
		return database.URUser{}, nil //todo maybe convert this to pointer so we can return nil
	}

	return user, nil
}

func SetSettings(settings Settings) error {

	_, err := database.DB.NewUpdate().Model(&settings).Where("discord_id = ?", settings.DiscordID).Exec(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func GetSettings(discordid string) (Settings, error) {
	settings := Settings{}

	err := database.DB.NewSelect().Model(&settings).Where("discord_id = ?", discordid).Limit(1).Scan(context.Background(), &settings)

	return settings, err
}

func GetOptedOutUsers() (users []string, err error) {

	f2, er2 := os.Open("out.json") //this is list of users who opted out of reviewdb
	if er2 != nil {
		fmt.Println(er2)
	}

	er2 = json.NewDecoder(f2).Decode(&users)

	f2.Close()

	userlist := []database.URUser{}

	err = database.DB.NewSelect().Model(&database.URUser{}).Where("opted_out = true").Scan(context.Background(), &userlist)

	for _, user := range userlist {
		users = append(users, user.DiscordID)
	}

	return
}

func GetReportCountInLastHour(userID int32) (int, error) {
	count, err := database.DB.
		NewSelect().Table("reports").
		Where("reporter_id = ? AND timestamp > now() - interval '1 hour'", userID).
		Count(context.Background())
	if err != nil {
		return 0, err
	}
	return count, nil
}
func AppealBan(appeal database.ReviewDBAppeal, user *database.URUser) (err error) {
	_, err = database.DB.NewInsert().Model(&appeal).Exec(context.Background())
	if err == nil {
		SendAppealWebhook(
			WebhookData{
				Username: "ReviewDB Appeals",
				Embeds: []Embed{
					{
						Title: "Appeal Form",
						Fields: []EmbedField{
							{
								Name:  "User",
								Value: formatUser(user.Username, user.ID, user.DiscordID),
							},
							{
								Name:  "Reason to appeal",
								Value: appeal.AppealText,
							},
						},
					},
				},
			},
		)
	}

	return
}
