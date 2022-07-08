package modules

import (
	"context"
	"encoding/json"
	"errors"

	"server-go/common"
	"server-go/database"
)

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

func AddReview(userID, token, comment string) (string, error) {
	senderUserID := GetIDWithToken(token)
	if senderUserID == 0 {
		return "", errors.New("invalid token")
	}
	count, _ := GetReviewCountInLastHour(senderUserID)
	if count > 20 {
		return "You are reviewing too much.", nil
	}

	var review database.UserReview
	review.UserID = userID
	review.SenderUserID = senderUserID
	review.Comment = comment
	review.Star = -1

	exists, _ := database.DB.
		NewSelect().
		Where("userid = ? AND senderuserid = ?", userID, senderUserID).
		Model((*database.UserReview)(nil)).
		Exists(context.Background())
	if exists {
		_, err := database.DB.NewUpdate().Where("userid = ? AND senderuserid = ?", userID, senderUserID).Model(&review).Exec(context.Background())
		if err != nil {
			return "An Error occurred while updating your review", err
		}
		return "Updated your review", nil
	}

	_, err := database.DB.NewInsert().Model(&review).Exec(context.Background())
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
	token, err := ExchangeCodePlus(code, common.GetConfig().Origin+"/URauth")
	if err != nil {
		return "", err
	}
	discordUser, err := GetUser(token)
	if err != nil {
		return "", err
	}

	var user database.URUser
	user.DiscordID = discordUser.ID
	user.Token = CalculateHash(token)
	user.Username = discordUser.Username + "#" + discordUser.Discriminator

	exists, _ := database.DB.
		NewSelect().
		Where("discordid = ?", discordUser.ID).
		Model((*database.URUser)(nil)).
		Exists(context.Background())
	if exists {
		_, err = database.DB.NewUpdate().Where("discordid = ?", discordUser.ID).Model(&user).Exec(context.Background())
		if err != nil {
			return "", err
		}
		return token, nil
	}
	_, err = database.DB.NewInsert().Model(&user).Exec(context.Background())
	if err != nil {
		return "An Error occurred", err
	}
	return token, nil
}
