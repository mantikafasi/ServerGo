package modules

import (
	"context"
	"encoding/json"
	"github.com/uptrace/bun"
)

type UserReviewStr struct {
	bun.BaseModel `bun:"table:userreviews,"`

	ID           int32  `bun:"id," json:"id"`
	UserID       int64  `bun:"userid," json:"userid"`
	SenderUserId int32  `bun:"senderuserid," json:"senderuserid"`
	Comment      string `bun:"comment" json:"comment"`
}

type UR_UserStr struct {
	bun.BaseModel `bun:"table:ur_users,"`
	ID            int32  `bun:"id," json:"id"`
	DiscordID     int64  `bun:"discordid," json:"discordid"`
	Token         string `bun:"token," json:"token"`
}

func GetReviews(DB *bun.DB, userID int64) (string, error) {
	var reviews []UserReviewStr

	err := DB.NewSelect().Where("userid = ?", userID).Model(&reviews).Scan(context.Background())
	if err != nil {
		return "bad error", err
	}
	//converts users to json string
	jsonReviews, err := json.Marshal(reviews)

	return string(jsonReviews), nil
}

func AddReview(DB *bun.DB, userID int64, token string, comment string) (string,error) {
	senderUserID := GetIDWithToken(DB, token)
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
	DB.NewSelect().Where("token = ?", token).Model(&userID).Scan(context.Background())

	return userID.ID
}

func GetReviewCountInLastHour(DB *bun.DB, userID int32) (int, error) {
	count,err := DB.NewSelect().Where("userid = ? AND createdat > now() - interval '1 hour'", userID).Count(context.Background())
	if err != nil {
		return 0, err
	}
	return count, nil
}

func AddUser(DB *bun.DB, discordID int64, token string) (string, error) {
	var user UR_UserStr
	user.DiscordID = discordID
	user.Token = token
	_,err := DB.NewInsert().Model(&user).Exec(context.Background())
	if err != nil {
		return "An Error Occured",err
	}
	return "Successfully added user", nil
}