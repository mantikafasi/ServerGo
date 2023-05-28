package modules

import (
	"context"
	"errors"
	"fmt"
	"server-go/common"
	"server-go/database"
	"server-go/database/schemas"
	"server-go/modules"

	"github.com/patrickmn/go-cache"
)

func GetTwitterReviews(userID string, offset int) ([]schemas.TwitterUserReview, int, error) {
	var reviews []schemas.TwitterUserReview

	count, err := database.DB.NewSelect().
		Model(&reviews).
		Relation("User").
		Where("profile_id = ?", userID).
		OrderExpr("id desc").Limit(51).
		Offset(offset).
		ScanAndCount(context.Background(), &reviews)

	if err != nil {
		return nil, 0, err
	}

	for i, review := range reviews {
		dbBadges := GetBadgesOfUser(review.User.TwitterID)
		badges := make([]schemas.TwitterUserBadge, len(dbBadges))
		for i, b := range dbBadges {
			badges[i] = schemas.TwitterUserBadge(b)
		}

		if review.User != nil {
			reviews[i].Sender.TwitterID = review.User.TwitterID
			reviews[i].Sender.AvatarURL = review.User.AvatarURL
			reviews[i].Sender.Username = review.User.Username
			reviews[i].Sender.ID = review.User.ID
			reviews[i].Sender.Badges = badges
		}
		reviews[i].Timestamp = review.TimestampStr.Unix()
	}

	return reviews, count, nil
}

func GetBadgesOfUser(discordid string) []schemas.TwitterUserBadge {
	userBadges := []schemas.TwitterUserBadge{}

	badges, _ := GetAllBadges()
	for _, badge := range badges {

		if badge.TargetTwitterID == discordid {
			userBadges = append(userBadges, badge)
		}
	}
	return userBadges
}

func GetAllBadges() (badges []schemas.TwitterUserBadge, err error) {

	cachedBadges, found := common.Cache.Get("twitterBadges")
	if found {
		badges = cachedBadges.([]schemas.TwitterUserBadge)
		return
	}

	badges = []schemas.TwitterUserBadge{}
	err = database.DB.NewSelect().Model(&badges).Scan(context.Background(), &badges)

	users := []schemas.TwitterUser{}

	database.DB.NewSelect().Distinct().Model(&users).Column("discord_id", "type").Where("type = ? or type = ?", 1, -1).Scan(context.Background(), &users)

	for _, user := range users {
		if user.Type == 1 {
			badges = append(badges, schemas.TwitterUserBadge{
				TargetTwitterID: user.TwitterID,
				Name:            "Admin",
				Icon:            "https://cdn.discordapp.com/emojis/1040004306100826122.gif?size=128",
				RedirectURL:     "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
				Description:     "This user is an admin of ReviewDB.",
			})
		} else {
			badges = append(badges, schemas.TwitterUserBadge{
				TargetTwitterID: user.TwitterID,
				Name:            "Banned",
				Icon:            "https://cdn.discordapp.com/emojis/399233923898540053.gif?size=128",
				RedirectURL:     "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
				Description:     "This user is banned from ReviewDB.",
			})
		}

	}

	common.Cache.Set("twitterBadges", badges, cache.DefaultExpiration)
	return
}

func GetReviewCountInLastHour(userID string) (int, error) {
	//return 0, nil
	count, err := database.DB.
		NewSelect().Table("reviewdb_twitter.reviews").
		Where("reviewer_id = ? AND timestamp > now() - interval '1 hour'", userID).
		Count(context.Background())
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetDBUserViaTwitterID(twitterID string) (*schemas.TwitterUser, error) {
	var user schemas.TwitterUser
	err := database.DB.NewSelect().Model(&user).Where("twitter_id = ?", twitterID).Limit(1).Scan(context.Background())

	if err != nil {
		if err.Error() == "sql: no rows in result set" { //SOMEONE TELL ME BETTER WAY TO DO THIS
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func GetDBUserViaToken(token string) (*schemas.TwitterUser, error) {
	var user schemas.TwitterUser
	err := database.DB.NewSelect().Model(&user).Where("token = ?", token).Limit(1).Scan(context.Background())

	if err != nil {
		if err.Error() == "sql: no rows in result set" { //SOMEONE TELL ME BETTER WAY TO DO THIS
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func AddTwitterUser(code string, ip string) (string, error) {
	twitterToken, err := ExchangeCode(code)
	if err != nil {
		return "", err
	}

	twitterUser, err := FetchUser(twitterToken.AccessToken)
	if err != nil {
		return "", err
	}
	token := modules.GenerateToken()

	user := &schemas.TwitterUser{
		TwitterID:    twitterUser.Data.ID,
		Token:        token,
		Username:     twitterUser.Data.Username,
		DisplayName:  twitterUser.Data.Name,
		AvatarURL:    twitterUser.Data.AvatarURL,
		Type:         0,
		IpHash:       modules.CalculateHash(ip),
		RefreshToken: twitterToken.RefreshToken,
		ExpiresAt:    twitterToken.Expiry,
	}

	dbUser, err := GetDBUserViaTwitterID(twitterUser.Data.ID)

	if dbUser != nil {
		if dbUser.Type == -1 {
			return "You have been banned from ReviewDB", errors.New("You have been banned from ReviewDB")
		}

		dbUser.Username = twitterUser.Data.Username
		dbUser.DisplayName = twitterUser.Data.Name
		dbUser.AvatarURL = twitterUser.Data.AvatarURL

		_, err = database.DB.NewUpdate().Where("id = ?", dbUser.ID).Model(dbUser).Exec(context.Background())
		if err != nil {
			return "", err
		}

		return dbUser.Token, nil
	}

	_, err = database.DB.NewInsert().Model(user).Exec(context.Background())
	if err != nil {
		return common.ERROR, err
	}

	modules.SendLoggerWebhook(modules.WebhookData{
		Username:  twitterUser.Data.Username,
		AvatarURL: twitterUser.Data.AvatarURL,
		Content:   fmt.Sprintf("User %s (%s) has been registered to ReviewDB Twitter", twitterUser.Data.Username, twitterUser.Data.ID),
	})
	return token, nil
}

func AddReview(user *schemas.TwitterUser, data schemas.TwitterRequestData) (response string, err error) {

	if common.LightProfanityDetector.IsProfane(data.Comment) || common.ProfanityDetector.IsProfane(data.Comment) {
		return "", errors.New("Your review contains profanity")
	}

	count, err := GetReviewCountInLastHour(user.TwitterID)

	if err != nil {
		println(err.Error())
		return "", errors.New(common.ERROR)
	}

	if count > 20 {
		return "", errors.New("You are reviewing too much")
	}

	review := schemas.TwitterUserReview{
		Comment:    data.Comment,
		ProfileID:  data.ProfileID,
		ReviewerID: user.TwitterID,
	}

	res, err := database.DB.NewUpdate().Where("profile_id = ? AND reviewer_id = ?", data.ProfileID, user.TwitterID).OmitZero().Model(review).Exec(context.Background())
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
