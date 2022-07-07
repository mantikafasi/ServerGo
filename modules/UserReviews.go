package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/uptrace/bun"
	"server-go/common"
)

type UserReviewStr struct {
	bun.BaseModel   `bun:"table:userreviews,"`
	ID              int32  `bun:"id," json:"id"`
	UserID          int64  `bun:"userid," json:"userid"`
	Star            int32  `bun:"star," json:"star"`
	SenderUserId    int32  `bun:"senderuserid," json:"senderuserid"`
	Comment         string `bun:"comment," json:"comment"`
	SenderDiscordID int64  `bun:"senderdiscordid," json:"senderdiscordid"`
	SenderUsername  string `bun:"username," json:"username"`
}

type BasicUserReviewStr struct {
	bun.BaseModel `bun:"table:userreviews,"`
	ID            int32  `bun:"id,pk,autoincrement" json:"id"`
	UserID        int64  `bun:"userid," json:"userid"`
	SenderUserId  int32  `bun:"senderuserid," json:"senderuserid"`
	Star          int32  `bun:"star," json:"star"`
	Comment       string `bun:"comment," json:"comment"`
}

type UR_UserStr struct {
	bun.BaseModel `bun:"table:ur_users,"`
	ID            int32  `bun:"id,pk,autoincrement" json:"id"`
	DiscordID     int64  `bun:"discordid," json:"discordid"`
	Token         string `bun:"token," json:"token"`
	Username      string `bun:"username," json:"username"`
}

func GetReviews(userID int64) (string, error) {
	DB := common.GetDB()

	var reviews []UserReviewStr

	err := DB.NewSelect().TableExpr("userreviews as r").
		ColumnExpr("r.id,r.userid,r.comment,r.senderuserid").
		ColumnExpr("u.discordid as senderdiscordid,u.username").
		Join("JOIN ur_users as u on senderuserid = u.id").
		Where("r.userid = ?", userID).
		Scan(context.Background(), &reviews)

	if err != nil {
		return "very bad error occured", err
	}

	jsonReviews, _ := json.Marshal(reviews)

	return string(jsonReviews), nil
}

func AddReview(userID int64, token string, comment string) (string, error) {
	DB := common.GetDB()

	senderUserID := GetIDWithToken(token)
	if senderUserID == 0 {
		return "", fmt.Errorf("Invalid Token")
	}
	count, _ := GetReviewCountInLastHour(senderUserID)
	if count > 20 {
		return "You are reviewing too much.", nil
	}

	var review BasicUserReviewStr
	review.UserID = userID
	review.SenderUserId = senderUserID
	review.Comment = comment
	review.Star = -1

	exists, _ := DB.NewSelect().Where("userid = ? AND senderuserid = ?", userID, senderUserID).Model(&BasicUserReviewStr{}).Exists(context.Background())
	if exists {
		_, err := DB.NewUpdate().Where("userid = ? AND senderuserid = ?", userID, senderUserID).Model(&review).Exec(context.Background())
		if err != nil {
			return "An Error Occured while updating your review", err
		}
		return "Updated your review", nil
	}

	_, err := DB.NewInsert().Model(&review).Exec(context.Background())
	if err != nil {
		return "An Error Occured", err
	}
	return "Added your review", nil
}

func GetIDWithToken(token string) int32 {
	DB := common.GetDB()

	var userID UR_UserStr
	DB.NewSelect().Where("token = ?", CalculateHash(token)).Model(&userID).Scan(context.Background())

	return userID.ID
}

func GetReviewCountInLastHour(userID int32) (int, error) {
	DB := common.GetDB()

	count, err := DB.NewSelect().Where("userid = ? AND createdat > now() - interval '1 hour'", userID).Count(context.Background())
	if err != nil {
		return 0, err
	}
	return count, nil
}

func AddUserReviewsUser(code string) (string, error) {
	DB := common.GetDB()

	token, err := ExchangeCodePlus(code,"http://192.168.1.35/URauth")
	if err != nil {
		return "", err
	}
	discordUser, err := GetUser(token)
	if err != nil {
		return "", err
	}

	var user UR_UserStr
	user.DiscordID = discordUser.ID
	user.Token = CalculateHash(token)
	user.Username = discordUser.Username + "#" + discordUser.Discriminator

	exists, _ := DB.NewSelect().Where("discordid = ?", discordUser.ID).Model(&UR_UserStr{}).Exists(context.Background())
	if exists {
		_, err := DB.NewUpdate().Where("discordid = ?", discordUser.ID).Model(&user).Exec(context.Background())
		if err != nil {
			return "", err
		}
		return token, nil
	}
	_, er := DB.NewInsert().Model(&user).Exec(context.Background())
	if er != nil {
		return "An Error Occured", err
	}
	return token, nil

}
