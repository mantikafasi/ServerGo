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

	"golang.org/x/exp/slices"

	"server-go/common"
	"server-go/database"
	"server-go/database/schemas"

	discord_utils "server-go/modules/discord"
	"server-go/modules/github"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/patrickmn/go-cache"
	"github.com/uptrace/bun"
)

type UR_RequestData struct {
	DiscordID  discord.Snowflake `json:"userid"`
	Token      string            `json:"token"`
	ReviewID   int32             `json:"reviewid"`
	Comment    string            `json:"comment"`
	ReviewType int               `json:"reviewtype"`
	Sender     struct {
		Username     string `json:"username"`
		ProfilePhoto string `json:"profile_photo"`
		DiscordID    string `json:"discord_id"`
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

type GetReviewsOptions struct {
	IncludeReviewsById string
}

func GetReviews(userID int64, offset int) ([]schemas.UserReview, int, error) {
	return GetReviewsWithOptions(userID, offset, GetReviewsOptions{
		IncludeReviewsById: "",
	})
}

func GetReviewsWithOptions(userID int64, offset int, options GetReviewsOptions) ([]schemas.UserReview, int, error) {
	var reviews []schemas.UserReview

	req := database.DB.NewSelect().
		Model(&reviews).
		Relation("User").
		Where("profile_id = ?", userID).
		Where("\"user\".\"opted_out\" = 'f'").
		Offset(offset).
		Limit(51)

	if options.IncludeReviewsById != "" {
		req = req.OrderExpr("reviewer_id = ? desc ,\"user\".discord_id = ? desc , id desc", options.IncludeReviewsById, options.IncludeReviewsById)
	} else {
		req = req.OrderExpr("id desc")
	}
	count, err := req.ScanAndCount(context.Background(), &reviews)
	if err != nil {
		return nil, 0, err
	}

	for i, review := range reviews {
		badges := GetBadgesOfUser(review.User.DiscordID)

		if review.Type == 4 {
			badges = append(badges, schemas.UserBadge{
				Name:        "StartIT",
				Icon:        "https://cdn.discordapp.com/attachments/1096421101132853369/1122886750763749488/logo-color.png?size=128",
				Description: "This review has been made by StartIT bot",
				RedirectURL: "https://startit.bot",
			})
		}

		if review.User.DiscordID == "1134864775000629298" {
			// troll
			reviews[i].Type = 3
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

func GetDBUserViaDiscordID(discordID string) (*schemas.URUser, error) {
	var user schemas.URUser
	err := database.DB.NewSelect().Model(&user).Where("discord_id = ?", discordID).Limit(1).Scan(context.Background())

	if err != nil {
		if err.Error() == "sql: no rows in result set" { //SOMEONE TELL ME BETTER WAY TO DO THIS
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func SearchReviews(query string, token string) ([]schemas.UserReview, error) {
	var reviews []schemas.UserReview

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

func AddReview(reviewer *schemas.URUser, review *schemas.UserReview) (string, error) {
	var err error

	res, err := database.DB.NewUpdate().Where("profile_id = ? AND reviewer_id = ?", review.ProfileID, reviewer.ID).OmitZero().Model(review).Exec(context.Background())
	if err != nil {
		return common.UPDATE_FAILED, err
	}

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
		Model((*schemas.URUser)(nil)).
		Column("id").
		Where("token = ? or token = ?", CalculateHash(token), token).
		Scan(context.Background(), &id)
	return
}

func GetDBUserViaTokenAndData(token string, data UR_RequestData) (user schemas.URUser, err error) {

	if token == common.Config.StartItBotToken {
		user, err := GetDBUserViaDiscordID(data.Sender.DiscordID)
		if err != nil {
			return schemas.URUser{}, err
			// todo sometime change retrun value to pointer
		}

		if user == nil {
			reviewer, err := CreateUserViaBot(data.Sender.DiscordID, data.Sender.Username, data.Sender.ProfilePhoto)
			if err != nil {
				return schemas.URUser{}, err
			}

			return reviewer, nil
		} else {
			return *user, nil
		}
	}

	err = database.DB.
		NewSelect().
		Model(&user).
		Where("token = ? or token = ?", token, CalculateHash(token)).
		Relation("BanInfo", func(sq *bun.SelectQuery) *bun.SelectQuery {
			// this does absolutely nothing but hoping that they will update bun this should work
			// https://github.com/uptrace/bun/issues/554
			return sq.JoinOn("join on ban_info.ban_end_date > now()").Order("ban_info.ban_end_date desc")
		}).
		Relation("Notification", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.JoinOn("join on notification.read = false").Order("notification.id desc")
		}).
		Scan(context.Background(), &user)

	if user.BanInfo != nil && user.BanInfo.BanEndDate.Before(time.Now()) {
		user.BanInfo = nil
	}

	if user.Notification != nil && user.Notification.Read {
		user.Notification = nil
	}

	if err != nil {
		fmt.Println(err.Error())
		return schemas.URUser{}, errors.New("Invalid Token")
	}

	// err = database.DB.NewSelect().
	// 	Model(&user).
	// 	Join("LEFT OUTER JOIN user_bans AS ban_info").JoinOn("ban_info.discord_id = ur_user.discord_id AND ban_info.ban_end_date > now()").
	// 	Join("LEFT OUTER JOIN notifications AS notification").JoinOn("notification.user_id = ur_user.id AND notification.read = false").
	// 	Where("token = ? or token = ?", token, CalculateHash(token)).
	// 	Scan(context.Background(), &user)

	// var notification *schemas.Notification
	// var banInfo *schemas.ReviewDBBanLog

	// response := &struct {
	// 	*schemas.URUser
	// 	*schemas.ReviewDBBanLog
	// 	*schemas.Notification
	// }{
	// 	&user,
	// 	banInfo,
	// 	notification,
	// }

	// err = database.DB.NewRaw(
	// 	`SELECT users.id as id,users.discord_id as discord_id,* FROM users
	// 	LEFT OUTER JOIN user_bans b ON b.discord_id = users.discord_id and b.ban_end_date > now()
	// 	LEFT OUTER JOIN notifications n ON n.user_id = users.id and n.read = false
	// 	WHERE token = ? or token = ?`,
	// 	CalculateHash(token), token).Scan(context.Background(), response)

	// guh := database.DB.DB.QueryRowContext(context.Background(), `SELECT users.id as id,users.discord_id as discord_id,* FROM users
	// 	LEFT OUTER JOIN user_bans b ON b.discord_id = users.discord_id and b.ban_end_date > now()
	// 	LEFT OUTER JOIN notifications n ON n.user_id = users.id and n.read = false
	// 	WHERE token = ? or token = ?`, CalculateHash(token), token)

	// err = guh.Scan(&user, &banInfo, &notification)

	// user.Notification = notification
	// user.BanInfo = banInfo

	return user, err
}

func GetDBUserViaToken(token string) (user schemas.URUser, err error) {
	return GetDBUserViaTokenAndData(token, UR_RequestData{})
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
	discordToken, err := discord_utils.ExchangeCode(code, common.Config.Origin+authUrl)
	if err != nil {
		fmt.Println(err)
		return "", errors.New("Invalid Code")
	}

	discordUser, err := discord_utils.GetUser(discordToken.AccessToken)

	if err != nil {
		fmt.Println(err)
		return "", errors.New(common.ERROR)
	}

	if discordUser.CreatedAt().After(time.Now().Add(-time.Hour * 24 * 30)) {
		return "", errors.New("Your account is too new")
	}

	token := GenerateToken()

	user := &schemas.URUser{
		DiscordID:    discordUser.ID.String(),
		Token:        token,
		Username:     common.Ternary(discordUser.Discriminator == "0", discordUser.Username, discordUser.Username+"#"+discordUser.Discriminator),
		AvatarURL:    discordUser.AvatarURL(),
		Type:         0,
		ClientMods:   []string{clientmod},
		IpHash:       CalculateHash(ip),
		AccessToken:  discordToken.AccessToken,
		RefreshToken: discordToken.RefreshToken,
	}
	if discordUser.Discriminator == "0" {
		user.Username = discordUser.Username
	}

	dbUser, err := GetDBUserViaDiscordID(discordUser.ID.String())

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

		dbUser.AccessToken = discordToken.AccessToken
		dbUser.RefreshToken = discordToken.RefreshToken
		dbUser.Username = user.Username
		dbUser.AvatarURL = discordUser.AvatarURL()

		_, err = database.DB.NewUpdate().Where("id = ?", dbUser.ID).Model(dbUser).Exec(context.Background())
		if err != nil {
			return "", err
		}

		return dbUser.Token, nil
	}

	_, err = database.DB.NewInsert().Model(user).Exec(context.Background())
	if err != nil {
		fmt.Println(err)
		return "An Error occurred", errors.New(common.ERROR)
	}

	discord_utils.SendLoggerWebhook(discord_utils.WebhookData{
		Username:  discordUser.Username + "#" + discordUser.Discriminator,
		AvatarURL: discordUser.AvatarURL(),
		Content:   fmt.Sprintf("User <@%s> has been registered to ReviewDB from %s", discordUser.ID, clientmod),
	})

	return token, nil
}

func GetReview(id int32) (rep schemas.UserReview, err error) {
	rep = schemas.UserReview{}
	err = database.DB.NewSelect().Model(&rep).Relation("User").Where("user_review.id = ?", id).Scan(context.Background(), &rep)
	if err != nil {
		return rep, err
	}

	badges := GetBadgesOfUser(rep.User.DiscordID)

	if rep.User != nil {
		rep.Sender = schemas.Sender{}
		rep.Sender.DiscordID = rep.User.DiscordID
		rep.Sender.ProfilePhoto = rep.User.AvatarURL
		rep.Sender.Username = rep.User.Username
		rep.Sender.ID = rep.User.ID
		rep.Sender.Badges = badges
	}
	rep.Timestamp = rep.TimestampStr.Unix()

	return
}

func ReportReview(data UR_RequestData) error {

	user, err := GetDBUserViaTokenAndData(data.Token, data)
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

	count, _ := database.DB.NewSelect().Model(&schemas.ReviewReport{}).Where("review_id = ? AND reporter_id = ?", data.ReviewID, user.ID).Count(context.Background())
	if count > 0 {
		return errors.New("You have already reported this review")
	}

	review, err := GetReview(data.ReviewID)
	if err != nil {
		return errors.New("Invalid Review ID")
	}

	if review.Sender.DiscordID == user.DiscordID {
		return errors.New("You cant report your own reviews")
	}

	reportedUser, _ := GetDBUserViaID(review.ReviewerID)

	report := schemas.ReviewReport{
		ReviewID:   data.ReviewID,
		ReporterID: user.ID,
	}

	err = discord_utils.SendReportWebhook(&user, &review, &reportedUser)

	if err != nil {
		println(err.Error())
	}

	database.DB.NewInsert().Model(&report).Exec(context.Background())
	return nil
}

func GetReports() (reports []schemas.ReviewReport, err error) {
	reports = []schemas.ReviewReport{}
	err = database.DB.NewSelect().Model(&reports).Scan(context.Background(), &reports)
	return
}

// checks if user is admin **or** moderator
func IsUserAdminDC(discordid int64) bool {
	count, _ := database.DB.NewSelect().Model(&schemas.URUser{}).Where("discord_id = ? and (type = 1 or type = 2)", discordid).Count(context.Background())

	if count > 0 {
		return true
	}
	return false
}

func GetDBUserViaID(id int32) (user schemas.URUser, err error) {
	user = schemas.URUser{}
	err = database.DB.NewSelect().Model(&user).Where("ur_user.id = ?", id).Relation("BanInfo").Scan(context.Background(), &user)
	if user.BanInfo != nil && user.BanInfo.BanEndDate.Before(time.Now()) {
		user.BanInfo = nil
	}
	return
}

func DeleteReview(reviewID int32, token string) (err error) {
	data := UR_RequestData{
		ReviewID: reviewID,
		Token:    token,
	}
	return DeleteReviewWithData(data)
}

func DeleteReviewWithData(data UR_RequestData) (err error) {
	review, err := GetReview(data.ReviewID)
	if err != nil {
		fmt.Println(err.Error())
		return errors.New("Invalid Review ID")
	}

	user, err := GetDBUserViaTokenAndData(data.Token, data)

	if err != nil && data.Token != common.Config.AdminToken { // todo create a admin account on database and handle things that way
		println(err.Error())
		return errors.New("Invalid Token")
	}

	if (review.User.DiscordID == user.DiscordID) || user.IsAdmin() || data.Token == common.Config.AdminToken || user.DiscordID == strconv.FormatInt(review.ProfileID, 10) {
		LogAction("DELETE", review, user.ID)

		_, err = database.DB.NewDelete().Model(&review).Where("id = ?", data.ReviewID).Exec(context.Background())
		return nil
	}
	return errors.New("You are not allowed to delete this review")
}

func GetBadgesOfUser(discordid string) []schemas.UserBadge {
	userBadges := []schemas.UserBadge{}

	badges, _ := GetAllBadges()
	for _, badge := range badges {

		if badge.TargetDiscordID == discordid {
			userBadges = append(userBadges, badge)
		}
	}
	return userBadges
}

func GetAllBadges() (badges []schemas.UserBadge, err error) {

	cachedBadges, found := common.Cache.Get("badges")
	if found {
		badges = cachedBadges.([]schemas.UserBadge)
		return
	}

	badges = []schemas.UserBadge{}
	err = database.DB.NewSelect().Model(&badges).Scan(context.Background(), &badges)

	users := []schemas.URUser{}

	database.DB.NewSelect().Distinct().Model(&users).Column("discord_id", "type").Where("type = ? or type = ?", 1, -1).Scan(context.Background(), &users)

	for _, user := range users {
		if user.Type == 1 {
			badges = append(badges, schemas.UserBadge{
				TargetDiscordID: user.DiscordID,
				Name:            "Admin",
				Icon:            "https://cdn.discordapp.com/emojis/1040004306100826122.gif?size=128",
				RedirectURL:     "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
				Description:     "This user is an admin of ReviewDB.",
			})
		} else {
			badges = append(badges, schemas.UserBadge{
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
	return database.DB.NewSelect().Model(&schemas.URUser{}).Count(context.Background())
}

func GetReviewCount() (count int, err error) {
	return database.DB.NewSelect().Model(&schemas.UserReview{}).Count(context.Background())
}

func GetLastReviewID(userID string) int32 {
	review := schemas.UserReview{}

	err := database.DB.NewSelect().Model(&schemas.UserReview{}).Where("profile_id = ?", userID).Order("id DESC").Limit(1).Scan(context.Background(), &review)

	if err != nil {
		return 0
	}

	return review.ID
}

func BanUser(userToBan string, adminToken string, banDuration int32, review schemas.UserReview) error {
	user := schemas.URUser{}

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

	if user.IsBanned() {
		return errors.New("This user is already banned")
	}

	if user.WarningCount >= 3 {
		_, err := database.DB.NewUpdate().Model(&schemas.URUser{}).Where("discord_id = ?", userToBan).Set("type = -1").Exec(context.Background())
		if err != nil {
			return err
		}
		return nil
	}

	var banData schemas.ReviewDBBanLog

	if review.ID != 0 {
		banData = schemas.ReviewDBBanLog{
			DiscordID:       userToBan,
			ReviewID:        review.ID,
			BanEndDate:      time.Now().AddDate(0, 0, int(banDuration)),
			ReviewContent:   review.Comment,
			ReviewTimestamp: review.TimestampStr,
		}

	} else {
		banData = schemas.ReviewDBBanLog{
			DiscordID:  userToBan,
			BanEndDate: time.Now().AddDate(0, 0, int(banDuration)),
		}
	}

	_, err := database.DB.NewInsert().Model(&banData).Exec(context.Background())
	if err != nil {
		return err
	}

	_, err = database.DB.NewUpdate().Model(&schemas.URUser{}).Where("discord_id = ?", userToBan).Set("ban_id = ?", banData.ID).Set("warning_count = warning_count + 1").Exec(context.Background())

	if err != nil {
		return err
	}

	SendNotification(&schemas.Notification{
		UserID: user.ID,
		Title:  "You have been banned from ReviewDB",
		Type:   schemas.NotificationTypeBan,
		Content: fmt.Sprintf(`
			You have been banned from ReviewDB %s

			**Offending Review:** %s

			Continued offenses will result in a permanent ban.
		`,
			common.Ternary(user.Type == schemas.UserTypeBanned, "permanently", "until <t:"+strconv.FormatInt(banData.BanEndDate.Unix(), 10)+":F>"),
			review.Comment,
		),
	})
	return nil
}

func GetAdmins() (users []string, err error) {
	users = []string{}
	userlist := []schemas.AdminUser{}

	err = database.DB.NewSelect().Distinct().Model(&schemas.AdminUser{}).Where("type = 1").Scan(context.Background(), &userlist)

	for _, user := range userlist {
		users = append(users, user.DiscordID)
	}
	return
}

func LogAction(action string, review schemas.UserReview, userid int32) {
	log := schemas.ActionLog{}

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

func CreateUserViaBot(discordid string, username string, profilePhoto string) (schemas.URUser, error) {
	user := schemas.URUser{}

	user.DiscordID = discordid
	user.Username = username
	user.Type = 0
	user.WarningCount = 0
	user.ClientMods = []string{"startitbot"}
	user.AvatarURL = profilePhoto
	user.Token = GenerateToken()

	_, err := database.DB.NewInsert().Model(&user).Exec(context.Background())
	if err != nil {
		println(err.Error())
		return schemas.URUser{}, errors.New("An Error Occured") //todo maybe convert this to pointer so we can return nil
	}

	discord_utils.SendLoggerWebhook(discord_utils.WebhookData{
		Username:  username,
		AvatarURL: profilePhoto,
		Content:   fmt.Sprintf("User <@%s> has been registered to ReviewDB from StartIT Bot", discordid),
	})

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

	userlist := []schemas.URUser{}

	err = database.DB.NewSelect().Model(&schemas.URUser{}).Where("opted_out = true").Scan(context.Background(), &userlist)

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

func AppealBan(appeal schemas.ReviewDBAppeal, user *schemas.URUser) (err error) {
	_, err = database.DB.NewInsert().Model(&appeal).Exec(context.Background())

	if err == nil {
		discord_utils.SendAppealWebhook(&appeal, user)
	}

	return
}

func GetAppeal(id int32) (appeal schemas.ReviewDBAppeal, err error) {
	database.DB.NewSelect().Model(&appeal).Where("id = ?", id).Scan(context.Background(), &appeal)
	return
}

func AcceptAppeal(appeal *schemas.ReviewDBAppeal, userId int32) (err error) {
	_, err = database.DB.NewUpdate().Model(&schemas.URUser{}).Set("type = 0").Set("ban_id = NULL").Set("warning_count = GREATEST(0, warning_count - 1)").Where("id = ?", userId).Exec(context.Background())

	if err != nil {
		return
	}

	_, err = database.DB.NewUpdate().Model(appeal).Set("action_taken=true").Exec(context.Background())

	err = SendNotification(&schemas.Notification{
		UserID:  userId,
		Title:   "ReviewDB",
		Content: "You have been unbanned from ReviewDB",
	})

	return
}

func DenyAppeal(appeal *schemas.ReviewDBAppeal, denyText string) (err error) {

	database.DB.NewUpdate().Model(appeal).Set("action_taken=true").Exec(context.Background())

	return SendNotification(&schemas.Notification{
		UserID: appeal.UserID,
		Title:  "ReviewDB",
		Content: fmt.Sprintf(`
			Your appeal has been denied
	
			**Reason:** %s,
		
		`, denyText),
	})
}

func GetBlockedUsers(blocker *schemas.URUser) (users []schemas.BaseRDBUser, err error) {
	if blocker.BlockedUsers == nil {
		return []schemas.BaseRDBUser{}, nil
	}

	users = []schemas.BaseRDBUser{}
	err = database.DB.NewSelect().Model(&users).Where("discord_id IN (?)", bun.In(blocker.BlockedUsers)).Scan(context.Background(), &users)
	return
}

func BlockUser(blocker *schemas.URUser, discordID string) (err error) {
	if len(blocker.BlockedUsers) > 50 {
		return errors.New("You can block maximum 50 users")
	}

	_, err = database.DB.NewUpdate().Model(&schemas.URUser{}).Set("blocked_users = array_append(blocked_users, ?)", discordID).Where("id = ?", blocker.ID).Exec(context.Background())
	return
}

func UnblockUser(blocker *schemas.URUser, discordID string) (err error) {
	_, err = database.DB.NewUpdate().Model(&schemas.URUser{}).Set("blocked_users = array_remove(blocked_users, ?)", discordID).Where("id = ?", blocker.ID).Exec(context.Background())
	return
}

func LinkGithub(githubCode string, user *schemas.URUser) (err error) {
	token, err := github.ExchangeCode(githubCode)
	if err != nil {
		return
	}

	userInfo, err := github.GetUserInfo(token.AccessToken)

	if err != nil {
		return
	}

	tokenEntry := schemas.Oauth2Token{
		UserId:       user.ID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		Provider:     "github",
		Username:     userInfo.Login,
		Avatar:       userInfo.AvatarURL,
		ProviderId:   userInfo.NodeID,
	}

	_, err = database.DB.NewInsert().Model(&tokenEntry).Exec(context.Background())

	return
}

func ResetToken(discordId string) (err error) {
	_, err = database.DB.NewUpdate().Model(&schemas.URUser{}).Set("token = ?", GenerateToken()).Where("discord_id = ?", discordId).Exec(context.Background())
	return
}