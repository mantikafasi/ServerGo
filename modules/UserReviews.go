package modules

// TODO: test this
//Join it with users table and return username and discordid along with reviews

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/uptrace/bun"
)

type UserReviewStr struct {
	bun.BaseModel `bun:"table:userreviews,"`
	ID           int32  `bun:"id," json:"id"`
	UserID       int64  `bun:"userid," json:"userid"`
	SenderUserId int32  `bun:"senderuserid," json:"senderuserid"`
	Comment      string `bun:"comment," json:"comment"`
	SenderDiscordID int64 `bun:"senderdiscordid," json:"senderdiscordid"`
	SenderUsername string `bun:"username," json:"username"`
}

type UR_UserStr struct {
	bun.BaseModel `bun:"table:ur_users,"`
	ID            int32  `bun:"id," json:"id"`
	DiscordID     int64  `bun:"discordid," json:"discordid"`
	Token         string `bun:"token," json:"token"`
	Username	  string `bun:"username," json:"username"`
}

func GetReviews(DB *bun.DB, userID int64) (string, error) {
	var reviews []UserReviewStr

	
	err := DB.NewSelect().TableExpr("userreviews as r").
	Model(&reviews).
	ColumnExpr("r.id,r.userid,r.comment").
	ColumnExpr("u.discordid as senderdiscordid,u.username").
	Join("JOIN ur_users as u on senderuserid = u.id").
	Where("r.userid = ?", userID).
	Scan(context.Background())
	
	if err != nil {
		return "very bad error occured", err
	}

	jsonReviews, _ := json.Marshal(reviews)

	return string(jsonReviews), nil
}

func AddReview(DB *bun.DB, userID int64, token string, comment string) (string,error) {
	senderUserID := GetIDWithToken(DB, token)
	if(senderUserID == 0) {
		return "", fmt.Errorf("Invalid Token")
	}
	count,_ := GetReviewCountInLastHour(DB, senderUserID)
	if count > 20 {
		return "You are reviewing too much.",nil
	}
	
	var review UserReviewStr
	review.UserID = userID
	review.SenderUserId = senderUserID
	review.Comment = comment
	_,err := DB.NewInsert().Model(&review).Exec(context.Background())
	if err != nil {
		return "An Error Occured",err
	}
	return "Successfully added review", nil
}

func GetIDWithToken(DB *bun.DB, token string) int32 {
	var userID UR_UserStr
	DB.NewSelect().Where("token = ?", CalculateHash(token)).Model(&userID).Scan(context.Background())

	return userID.ID
}

func GetReviewCountInLastHour(DB *bun.DB, userID int32) (int, error) {
	count,err := DB.NewSelect().Where("userid = ? AND createdat > now() - interval '1 hour'", userID).Count(context.Background())
	if err != nil {
		return 0, err
	}
	return count, nil
}

func AddUserReviewsUser(DB *bun.DB, code string) (string, error) {

	token, err := ExchangeCode(code)
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
		// update user
		_, err := DB.NewUpdate().Where("discordid = ?", discordUser.ID).Model(&user).Exec(context.Background())
		if err != nil {
			return "", err
		}
		return token,nil
	}
	_,er := DB.NewInsert().Model(&user).Exec(context.Background())
	if er != nil {
		return "An Error Occured",err
	}
	return token, nil
	
}