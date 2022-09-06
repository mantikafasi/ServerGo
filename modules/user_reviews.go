package modules

import (
	"context"
	"encoding/json"
	"errors"

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
	token, err := ExchangeCodePlus(code, common.Config.Origin+"/URauth")
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

func ReportReview(reviewID int64,token string) (error) {
	//todo
	return nil
}

func GetReports() (error) {
	//todo
	return nil
}

func GetAuthorReviews(userid int64) (error) {
	//todo
	return nil
}


func DeleteReview(reviewID int64,reviewid string) (err error){
	//todo
	return nil
}