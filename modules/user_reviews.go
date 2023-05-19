package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
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

	DiscordID string `bun:"discord_id"`
	Opt       bool   `json:"opt" bun:"opted_out"`
}

func GetReviews(userID int64, offset int) ([]database.UserReview, error) {
	var reviews []database.UserReview

	err := database.DB.NewSelect().
		Model(&reviews).
		Relation("User").
		Where("profile_id = ?", userID).
		Where("\"user\".\"opted_out\" = 'f'").
		OrderExpr("ID DESC").Limit(51).
		Offset(offset).
		Scan(context.Background(), &reviews)
	if err != nil {
		return nil, err
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

	return reviews, nil
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
	if user.UserType != 1 {
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
	var senderUserID int32
	if data.Token == common.Config.StupidityBotToken {

		user, err := GetDBUserViaDiscordID(data.Sender.DiscordID)
		if err != nil {
			return "", err
		}

		if user == nil {
			err, senderUserID = CreateUserViaBot(data.Sender.DiscordID, data.Sender.Username, data.Sender.ProfilePhoto)
			if err != nil {
				return "", err
			}
		} else {
			senderUserID = user.ID
		}

	} else {
		senderUserID = GetIDWithToken(data.Token)
	}

	if senderUserID == 0 {
		return "", errors.New("invalid token")
	}

	user, _ := GetDBUserViaID(senderUserID)

	if len(user.Token) == 64 && len(data.Token) != 64 { // try to get rid of old token system as much as possible
		user.Token = data.Token
		database.DB.NewUpdate().Model(user).Set("token = ?", data.Token).Where("id = ?", user.ID).Exec(context.Background())
	}

	if user.OptedOut {
		return "", errors.New("You have opted out of ReviewDB")
	}

	if !(data.ReviewType == 0 || data.ReviewType == 1) && user.UserType != 1 {
		return "", errors.New("Invalid review type")
	}

	if user.UserType == -1 || user.WarningCount > 2 {
		return "", errors.New("You have been banned from ReviewDB")
	}

	if user.BanInfo != nil {
		return "", errors.New("You have been banned from ReviewDB until " + user.BanInfo.BanEndDate.Format("2006-01-02 15:04:05") + "UTC")
	}

	count, _ := GetReviewCountInLastHour(senderUserID)
	if count > 20 {
		return "", errors.New("You are reviewing too much")
	}

	if common.LightProfanityDetector.IsProfane(data.Comment) {
		return "", errors.New("Your review contains profanity")
	}

	review := &database.UserReview{

		ProfileID:    int64(data.DiscordID),
		ReviewerID:   senderUserID,
		Comment:      data.Comment,
		Type:         int32(data.ReviewType),
		TimestampStr: time.Now(),
	}

	if common.ProfanityDetector.IsProfane(data.Comment) {
		review.ID = -1
		BanUser(user.DiscordID, common.Config.AdminToken, 7, *review)
		SendLoggerWebhook(WebhookData{
			Username: "ReviewDB",
			Content:  "User <@" + user.DiscordID + "> has been banned for 1 week for trying to post a profane review",
			Embeds: []Embed{
				{
					Fields: []ReportWebhookEmbedField{
						{
							Name:  "Review Content",
							Value: data.Comment,
						},
						{
							Name:  "ReviewDB ID",
							Value: strconv.Itoa(int(senderUserID)),
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

	res, err := database.DB.NewUpdate().Where("profile_id = ? AND reviewer_id = ?", data.DiscordID, senderUserID).OmitZero().Model(review).Exec(context.Background())
	if err != nil {

		return "An Error occurred while updating your review", err
	}
	//LogAction("UPDATE",review,senderUserID) TODO : DO THIS

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return "An Error occurred while updating your review", err
	}
	if rowsAffected != 0 {
		return "Updated your review", nil
	}

	_, err = database.DB.NewInsert().Model(review).Exec(context.Background())
	if err != nil {
		return "An Error occurred", err
	}
	return "Added your review", nil
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
	token := discordToken.AccessToken

	user := &database.URUser{
		DiscordID:    discordUser.ID,
		Token:        token,
		Username:     discordUser.Username + "#" + discordUser.Discriminator,
		AvatarURL:    GetProfilePhotoURL(discordUser.ID, discordUser.Avatar),
		UserType:     0,
		ClientMods:   []string{clientmod},
		IpHash:       CalculateHash(ip),
		RefreshToken: discordToken.RefreshToken,
	}

	dbUser, err := GetDBUserViaDiscordID(discordUser.ID)

	if dbUser != nil {
		if dbUser.UserType == -1 {
			return "You have been banned from ReviewDB", errors.New("You have been banned from ReviewDB")
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
	return
}

func ReportReview(reviewID int32, token string) error {

	user, err := GetDBUserViaToken(token)
	if err != nil {
		return errors.New("Invalid Token, please reauthenticate")
	}

	if user.BanInfo != nil || user.UserType == -1 {
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
				Fields: []ReportWebhookEmbedField{
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

func IsUserAdmin(id int32) bool {
	user := database.URUser{}
	database.DB.NewSelect().Model(&user).Where("id = ?", id).Scan(context.Background(), &user)
	if user.UserType == 1 {
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

	if (review.User.DiscordID == user.DiscordID) || IsUserAdmin(user.ID) || token == common.Config.AdminToken {
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
		if user.UserType == 1 {
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

func BanUser(discordid string, token string, banDuration int32, review database.UserReview) error {
	users := []database.URUser{}

	if token != common.Config.AdminToken && !IsUserAdmin(GetIDWithToken(token)) {
		return errors.New("You are not allowed to ban users")
	}

	/*
		AdminDiscordID := 0
		if token != common.Config.AdminToken {
			admin , _ := GetDBUserViaToken(token)
			AdminDiscordID = admin.DiscordID
		}
	*/

	database.DB.NewSelect().Model(&users).Where("discord_id = ?", discordid).Scan(context.Background(), &users)

	for user := range users {
		if users[user].UserType == 1 {
			return errors.New("You can't ban an admin")
		}
		if users[user].WarningCount >= 3 {
			_, err := database.DB.NewUpdate().Model(&database.URUser{}).Where("discord_id = ?", discordid).Set("type = -1").Exec(context.Background())
			if err != nil {
				return err
			}
			return nil
		}
	}

	var banData database.ReviewDBBanLog

	if review.ID != 0 {
		banData = database.ReviewDBBanLog{
			DiscordID:       discordid,
			ReviewID:        review.ID,
			BanEndDate:      time.Now().AddDate(0, 0, int(banDuration)),
			ReviewContent:   review.Comment,
			ReviewTimestamp: review.TimestampStr,
		}

	} else {
		banData = database.ReviewDBBanLog{
			DiscordID:  discordid,
			BanEndDate: time.Now().AddDate(0, 0, int(banDuration)),
		}
	}

	database.DB.NewInsert().Model(&banData).Exec(context.Background())

	_, err := database.DB.NewUpdate().Model(&database.URUser{}).Where("discord_id = ?", discordid).Set("ban_id = ?", banData.ID).Set("warning_count = warning_count + 1").Exec(context.Background())

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

func CreateUserViaBot(discordid string, username string, profilePhoto string) (error, int32) {
	user := database.URUser{}

	user.DiscordID = discordid
	user.Username = username
	user.UserType = 0
	user.WarningCount = 0
	user.ClientMods = []string{"discordbot"}
	user.AvatarURL = profilePhoto
	user.Token = discordid

	_, err := database.DB.NewInsert().Model(&user).Exec(context.Background())
	if err != nil {
		return err, 0
	}

	database.DB.NewSelect().Model(&user).Where("discord_id = ?", discordid).Limit(1).Scan(context.Background(), &user)

	return nil, user.ID
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

	err = database.DB.NewSelect().Distinct().Model(&database.URUser{}).Where("opted_out = true").Scan(context.Background(), &userlist)

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
