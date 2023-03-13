package modules

import (
	"context"
	"errors"
	"fmt"
	"time"

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
}

type ReportData struct {
	ReviewID int32  `json:"reviewid"`
	Token    string `json:"token"`
}

type Sender struct {
	ID           int32       `json:"id"`
	DiscordID    string      `json:"discordID"`
	Username     string      `json:"username"`
	ProfilePhoto string      `json:"profilePhoto"`
	Badges       []UserBadge `json:"badges"`
}

type UserReview struct {
	bun.BaseModel `bun:"table:userreviews"`

	ID           int32     `bun:"id,pk,autoincrement" json:"id"`
	UserID       int64     `bun:"userid,type:numeric" json:"-"`
	Sender       Sender    `bun:"-" json:"sender"`
	Star         int32     `bun:"star" json:"star"`
	Comment      string    `bun:"comment" json:"comment"`
	ReviewType   int32     `bun:"reviewtype" json:"type"` // 0 = user review , 1 = server review , 2 = support review, 3 = system review
	TimestampStr time.Time `bun:"timestamp,default:current_timestamp" json:"-"`
	Timestamp    int64     `bun:"-" json:"timestamp"`

	User         *database.URUser `bun:"rel:belongs-to,join:senderuserid=id" json:"-"`
	SenderUserID int32            `bun:"senderuserid" json:"-"`
}

type UserBadge struct {
	bun.BaseModel `bun:"table:userbadges"`

	ID               int32  `bun:"id,pk,autoincrement" json:"-"`
	DiscordID        string `bun:"discordid,type:numeric" json:"-"`
	BadgeName        string `bun:"badge_name" json:"name"`
	BadgeIcon        string `bun:"badge_icon" json:"icon"`
	RedirectURL      string `bun:"redirect_url" json:"redirectURL"`
	BadgeType        int32  `bun:"badge_type" json:"type"`
	BadgeDescription string `bun:"badge_description" json:"description"`
}

func GetReviews(userID int64) ([]UserReview, error) {
	var reviews []UserReview

	err := database.DB.NewSelect().Model(&reviews).Relation("User").Where("userid = ?", userID).OrderExpr("ID DESC").Limit(50).Scan(context.Background(), &reviews)
	if err != nil {
		return nil, err
	}

	for i, review := range reviews {
		dbBadges := GetBadgesOfUser(review.User.DiscordID)
		badges := make([]UserBadge, len(dbBadges))
		for i, b := range dbBadges {
			badges[i] = UserBadge(b)
		}

		if review.User != nil {
			reviews[i].Sender.DiscordID = review.User.DiscordID
			reviews[i].Sender.ProfilePhoto = review.User.ProfilePhoto
			reviews[i].Sender.Username = review.User.Username
			reviews[i].Sender.ID = review.User.ID
			reviews[i].Sender.Badges = badges
		}
		reviews[i].Timestamp = review.TimestampStr.Unix()
	}

	return reviews, nil
}

func GetReviewsLegacy(userID int64) ([]database.UserReview, error) {
	var reviews []database.UserReview

	err := database.DB.NewSelect().Model(&reviews).Relation("User").Where("userid = ?", userID).OrderExpr("ID DESC").Limit(50).Scan(context.Background(), &reviews)
	if err != nil {
		return nil, err
	}

	for i, review := range reviews {
		if review.User != nil {
			reviews[i].SenderDiscordID = review.User.DiscordID
			reviews[i].ProfilePhoto = review.User.ProfilePhoto
			reviews[i].SenderUsername = review.User.Username
			reviews[i].Badges = GetBadgesOfUser(review.User.DiscordID)
		}
		reviews[i].Timestamp = review.TimestampStr.Unix()
	}

	return reviews, nil
}

func AddReview(userID Snowflake, token, comment string, reviewtype int32) (string, error) {

	senderUserID := GetIDWithToken(token)

	if senderUserID == 0 {
		return "", errors.New("invalid token")
	}

	user, _ := GetDBUserViaID(senderUserID)
	if user.UserType == -1 {
		return "", errors.New("You have been banned from ReviewDB")
	}

	count, _ := GetReviewCountInLastHour(senderUserID)
	if count > 20 {
		return "You are reviewing too much.", nil
	}

	if common.ProfanityDetector.IsProfane(comment) {
		return "Your review contains profanity.", nil
	}

	review := &database.UserReview{
		UserID:       int64(userID),
		SenderUserID: senderUserID,
		Comment:      comment,
		Star:         -1,
		ReviewType:   reviewtype,
	}

	res, err := database.DB.NewUpdate().Where("userid = ? AND senderuserid = ?", userID, senderUserID).Model(review).Exec(context.Background())
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
		Where("token = ?", CalculateHash(token)).
		Scan(context.Background(), &id)
	return
}

func GetReviewCountInLastHour(userID int32) (int, error) {
	//return 0, nil
	count, err := database.DB.
		NewSelect().Table("user_reviews").
		Where("userid = ? AND timestamp > now() - interval '1 hour'", userID).
		Count(context.Background())
	if err != nil {
		return 0, err
	}
	return count, nil
}

func AddUserReviewsUser(code string, clientmod string, authUrl string) (string, error) {
	//todo make this work exactly same as pyton version
	if authUrl == "" {
		authUrl = "/URauth"
	}
	token, err := ExchangeCodePlus(code, common.Config.Origin+authUrl)
	if err != nil {
		return "", err
	}

	discordUser, err := GetUser(token)
	if err != nil {
		return "", err
	}

	user := &database.URUser{
		DiscordID:    discordUser.ID,
		Token:        CalculateHash(token),
		Username:     discordUser.Username + "#" + discordUser.Discriminator,
		ProfilePhoto: GetProfilePhotoURL(discordUser.ID, discordUser.Avatar),
		UserType:     0,
		ClientMod:    clientmod,
	}

	banned, err := database.DB.NewSelect().Model(&database.URUser{}).Where("discordid = ? and type = -1", discordUser.ID).ScanAndCount(context.Background())

	if banned != 0 {
		return "You have been banned from ReviewDB", errors.New("You have been banned from ReviewDB") //this is pretty much useless since it doesnt returns errors but whatever
	}

	count, err := database.DB.NewSelect().Model(user).Where("discordid = ? and token = ?", discordUser.ID, CalculateHash(token)).ScanAndCount(context.Background())
	if count != 0 {
		return token, nil
	}

	res, err := database.DB.NewUpdate().Where("discordid = ? and client_mod = ?", discordUser.ID, clientmod).Model(user).Exec(context.Background())
	if err != nil {
		return "", err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return "", err
	}

	if rowsAffected != 0 {
		return token, nil
	}

	_, err = database.DB.NewInsert().Model(user).Exec(context.Background())
	if err != nil {
		return "An Error occurred", err
	}
	return token, nil
}

func GetReview(id int32) (rep database.UserReview, err error) {
	rep = database.UserReview{}
	err = database.DB.NewSelect().Model(&rep).Where("id = ?", id).Scan(context.Background(), &rep)
	return
}

func ReportReview(reviewID int32, token string) error {

	user, err := GetDBUserViaID(GetIDWithToken(token))
	if err != nil {
		return errors.New("Invalid Token, please reauthenticate")
	}

	count, _ := database.DB.NewSelect().Model(&database.ReviewReport{}).Where("reviewid = ? AND reporterid = ?", reviewID, user.ID).Count(context.Background())
	if count > 0 {
		return errors.New("You have already reported this review")
	}

	review, err := GetReview(reviewID)
	if err != nil {
		return errors.New("Invalid Review ID")
	}
	reportedUser, _ := GetDBUserViaID(review.SenderUserID)

	report := database.ReviewReport{
		UserID:     review.SenderUserID,
		ReviewID:   reviewID,
		ReporterID: user.ID,
	}

	err = SendReportWebhook(ReportWebhookData{
		Username: "ReviewDB",
		Content:  "Reported Reveiew",
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
						CustomID: fmt.Sprintf("ban_user:%s", string(reportedUser.DiscordID)),
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
						CustomID: fmt.Sprintf("delete_and_ban:%d:%s", reviewID, string(reportedUser.DiscordID)),
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
						Name:  "Reporter ID",
						Value: fmt.Sprint(user.ID),
					},
					{
						Name:  "Reporter Username",
						Value: fmt.Sprint(user.Username),
					},
					{
						Name:  "Reported User Username",
						Value: fmt.Sprint(reportedUser.Username),
					},
					{
						Name:  "Reported Review ID",
						Value: fmt.Sprint(review.ID),
					},
					{
						Name:  "Reported Review Content",
						Value: fmt.Sprint(review.Comment),
					},
					{
						Name:  "Reported User ID",
						Value: fmt.Sprint(reportedUser.ID),
					},
				},
			},
		},
	})

	if err != nil {
		println(err.Error())
	}

	database.DB.NewInsert().Model(&report).Exec(context.Background())
	return nil
}

func GetReports() (reports []database.ReviewReport, err error) {
	reports = []database.ReviewReport{}
	err = database.DB.NewSelect().Model(&reports).Scan(context.Background(), &reports)
	return
}

func IsUserAdminDC(discordid int64) bool {
	user := database.URUser{}
	database.DB.NewSelect().Model(&user).Where("discordid = ?", discordid).Scan(context.Background(), &user)
	if user.UserType == 1 {
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
	err = database.DB.NewSelect().Model(&user).Where("id = ?", id).Scan(context.Background(), &user)
	return
}

func DeleteReview(reviewID int32, token string) (err error) {
	review, err := GetReview(reviewID)
	if err != nil {
		return errors.New("Invalid Review ID")
	}
	userid := GetIDWithToken(token)


	if (review.SenderUserID == userid) || IsUserAdmin(userid) || token == common.Config.AdminToken {
		LogAction("DELETE",review,userid)

		_, err = database.DB.NewDelete().Model(&review).Where("id = ?", reviewID).Exec(context.Background())
		return nil
	}
	return errors.New("You are not allowed to delete this review")
}

func GetBadgesOfUser(discordid string) []database.UserBadge {
	userBadges := []database.UserBadge{}

	badges, _ := GetAllBadges()
	for _, badge := range badges {

		if badge.DiscordID == discordid {

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

	database.DB.NewSelect().Distinct().Model(&users).Column("discordid", "type").Where("type = ? or type = ?", 1, -1).Scan(context.Background(), &users)

	for _, user := range users {
		if user.UserType == 1 {
			badges = append(badges, database.UserBadge{
				DiscordID:        user.DiscordID,
				BadgeName:        "Admin",
				BadgeIcon:        "https://cdn.discordapp.com/emojis/1040004306100826122.gif?size=128",
				RedirectURL:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
				BadgeDescription: "This user is an admin of ReviewDB.",
			})
		} else {
			badges = append(badges, database.UserBadge{
				DiscordID:        user.DiscordID,
				BadgeName:        "Banned",
				BadgeIcon:        "https://cdn.discordapp.com/emojis/399233923898540053.gif?size=128",
				RedirectURL:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
				BadgeDescription: "This user is banned from ReviewDB.",
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

	err := database.DB.NewSelect().Model(&database.UserReview{}).Where("userid = ?", userID).Order("id DESC").Limit(1).Scan(context.Background(), &review)

	if err != nil {
		return 0
	}

	return review.ID
}

func BanUser(discordid string, token string) error {
	users := []database.URUser{}

	if !IsUserAdmin(GetIDWithToken(token)) && token != common.Config.AdminToken {
		return errors.New("You are not allowed to ban users")
	}

	database.DB.NewSelect().Model(&users).Where("discordid = ?", discordid).Scan(context.Background(), &users)

	for user := range users {
		if users[user].UserType == 1 {
			return errors.New("You can't ban an admin")
		}
		users[user].UserType = -1
	}

	_, err := database.DB.NewUpdate().Model(&database.URUser{}).Where("discordid = ?", discordid).Set("type = -1").Exec(context.Background())
	//_, err := database.DB.NewUpdate().Model(&users).Where("discordid = ?", discordid).Exec(context.Background())
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

	log.UserID = review.UserID
	log.Action = action
	log.ReviewID = review.ID
	log.SenderUserID = review.SenderUserID
	log.Comment = review.Comment
	log.ActionUserID = userid

	_, err := database.DB.NewInsert().Model(&log).Exec(context.Background())
	if err != nil {
		fmt.Println(err)
	}
}

