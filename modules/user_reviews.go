package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"server-go/common"
	"server-go/database"

	"github.com/patrickmn/go-cache"
)

type UR_RequestData struct {
	DiscordID  Snowflake  `json:"userid"`
	Token      string `json:"token"`
	Comment    string `json:"comment"`
	ReviewType int    `json:"reviewtype"`
}

type ReportData struct {
	ReviewID int32  `json:"reviewid"`
	Token    string `json:"token"`
}

func GetReviews(userID int64) (string, error) {
	//todo add ads trol
	var reviews []database.UserReview

	err := database.DB.NewSelect().Model(&reviews).Relation("User").Where("userid = ?", userID).Limit(50).Scan(context.Background(), &reviews)
	if err != nil {
		return "very bad error occurred", err
	}

	for i, review := range reviews {
		if review.User != nil {
			reviews[i].SenderDiscordID = review.User.DiscordID
			reviews[i].ProfilePhoto = review.User.ProfilePhoto
			reviews[i].SenderUsername = review.User.Username
			reviews[i].Badges = GetBadgesOfUser(review.User.DiscordID)
		}
	}
	jsonReviews, _ := json.Marshal(reviews)
	return string(jsonReviews), nil
}

func AddReview(userID Snowflake, token, comment string, reviewtype int32) (string, error) {

	senderUserID := GetIDWithToken(token)
	
	if senderUserID == 0 {
		return "", errors.New("invalid token")
	}

	user,_ := GetDBUserViaID(senderUserID)
	if (user.UserType == -1) {
		return "", errors.New("You have been banned from ReviewDB")
	}

	count, _ := GetReviewCountInLastHour(senderUserID)
	if count > 20 {
		return "You are reviewing too much.", nil
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

func AddUserReviewsUser(code string, clientmod string) (string, error) {
	//todo make this work exactly same as pyton version
	token, err := ExchangeCodePlus(code, common.Config.Origin +"/URauth")
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

	SendReportWebhook(ReportWebhookData{
		Content: "Reported Reveiew",
		Embeds: []ReportWebhookEmbed{
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

	if (review.SenderUserID == userid) || IsUserAdmin(userid) {
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
				DiscordID:   user.DiscordID,
				BadgeName:   "Admin",
				BadgeIcon:   "https://cdn.discordapp.com/emojis/1040004306100826122.gif?size=128",
				RedirectURL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			})
		} else {
			badges = append(badges, database.UserBadge{
				DiscordID:   user.DiscordID,
				BadgeName:   "Banned",
				BadgeIcon:   "https://cdn.discordapp.com/emojis/399233923898540053.gif?size=128",
				RedirectURL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
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
