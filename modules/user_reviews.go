package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"server-go/common"
	"server-go/database"
)

type UR_RequestData struct {
	DiscordID  int64  `json:"userid"`
	Token      string `json:"token"`
	Comment    string `json:"comment"`
	ReviewType int    `json:"reviewtype"`
}

func GetReviews(userID int64) (string, error) {
	var reviews []database.UserReview

	err := database.DB.NewSelect().Model(&reviews).Relation("User").Where("userid = ?", userID).Scan(context.Background(), &reviews)
	if err != nil {
		return "very bad error occurred", err
	}

	for i, review := range reviews {
		if review.User != nil {
			reviews[i].SenderDiscordID = review.User.DiscordID
			reviews[i].SenderUsername = review.User.Username
		}
		//todo add badges
	}
	jsonReviews, _ := json.Marshal(reviews)
	return string(jsonReviews), nil
}

func AddReview(userID int64, token, comment string, reviewtype int32) (string, error) {

	senderUserID := GetIDWithToken(token)
	if senderUserID == 0 {
		return "", errors.New("invalid token")
	}
	count, _ := GetReviewCountInLastHour(senderUserID)
	if count > 20 {
		return "You are reviewing too much.", nil
	}

	review := &database.UserReview{
		UserID:       userID,
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

func AddUserReviewsUser(code string) (string, error) {
	token, err := ExchangeCodePlus(code, common.Config.Origin + "/URauth")
	if err != nil {
		return "", err
	}

	discordUser, err := GetUser(token)
	if err != nil {
		return "", err
	}

	user := &database.URUser{
		DiscordID: discordUser.ID,
		Token:     CalculateHash(token),
		Username:  discordUser.Username + "#" + discordUser.Discriminator,
		UserType: 0,
	}

	res, err := database.DB.NewUpdate().Where("discordid = ?", discordUser.ID).Model(user).Exec(context.Background())
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

func GetReview(id int32) (rep database.UserReview,err error) {
	rep = database.UserReview{}
	err = database.DB.NewSelect().Model(&rep).Where("id = ?",id).Scan(context.Background(),&rep)
	return
}

func ReportReview(reviewID int32,token string) (error) {
	user ,err := GetDBUserViaID(GetIDWithToken(token))
	if err != nil {
		return errors.New("Invalid Token, please reauthenticate")
	}

	review,err := GetReview(reviewID)
	if err != nil {
		return err
	}
	reportedUser,_ := GetDBUserViaID(review.SenderUserID)


	report := database.ReviewReport{
		UserID: review.SenderUserID,
		ReviewID: review.ID,
		ReporterID: user.ID,
	}
	
	SendReportWebhook(ReportWebhookData{
		Content: "Reported Reveiew",
		Embeds: []ReportWebhookEmbed{
			ReportWebhookEmbed{
				Fields: []ReportWebhookEmbedField{
					ReportWebhookEmbedField{
						Name: "Reporter ID",
						Value: fmt.Sprint(user.ID),
					},
					ReportWebhookEmbedField{
						Name: "Reporter Username",
						Value: fmt.Sprint(user.Username),
					},
					ReportWebhookEmbedField{
						Name: "Reported User Username",
						Value: fmt.Sprint(reportedUser.Username),
					},
					ReportWebhookEmbedField{
						Name: "Reported Review ID",
						Value: fmt.Sprint(review.ID),
					},
					ReportWebhookEmbedField{
						Name: "Reported Review Content",
						Value: fmt.Sprint(review.Comment),
					},
					ReportWebhookEmbedField{
						Name: "Reported User ID",
						Value: fmt.Sprint(reportedUser.ID),
					},
				},

			},
		},
	})

	database.DB.NewInsert().Model(&report).Exec(context.Background())
	return nil
}

func GetReports() (reports []database.ReviewReport,err error) {
	reports = []database.ReviewReport{}
	err = database.DB.NewSelect().Model(&reports).Scan(context.Background(),&reports)
	return
}

func IsUserAdminDC(discordid int64) (bool) {
	user := database.URUser{}
	database.DB.NewSelect().Model(&user).Where("discordid = ?",discordid).Scan(context.Background(),&user)
	if user.UserType == 1 {
		return true
	}
	return false
}

func IsUserAdmin(id int32) (bool) {
	user := database.URUser{}
	database.DB.NewSelect().Model(&user).Where("id = ?",id).Scan(context.Background(),&user)
	if user.UserType == 1 {
		return true
	}
	return false
}

func GetDBUserViaID(id int32) (user database.URUser,err error) {
	user = database.URUser{}
	err = database.DB.NewSelect().Model(&user).Where("id = ?",id).Scan(context.Background(),&user)
	return
}

func DeleteReview(reviewID int32,token string) (err error){
	review,err := GetReview(reviewID)
	if err != nil {
		return errors.New("Invalid Review ID")
	}
	userid := GetIDWithToken(token)

	if (review.SenderUserID == userid) || IsUserAdmin(userid) {
		_,err = database.DB.NewDelete().Model(&review).Where("id = ?",reviewID).Exec(context.Background())
		return
	}
	return errors.New("You are not allowed to delete this review")
}

func GetBadgesOfUser(discordid int64) (error) {
	//todo
	return nil
}

func GetAllBadges() (error) {
	//todo
	return nil
}

func GetVencordBadges() (error) {
	//todo
	return nil
}

func AddBadge(discordid int64,badge_name string,badge_icon string,redirect_url string) (error) {
	//todo
	return nil
}

