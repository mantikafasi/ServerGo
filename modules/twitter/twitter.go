package modules

import (
	"context"
	"server-go/common"
	"server-go/database"
	"server-go/database/schemas"

	"github.com/patrickmn/go-cache"
)

func GetReviews(userID int64, offset int) ([]schemas.TwitterUserReview, int, error) {
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

/*
func AddReviewDBTwitterUser(code string, ip string) (string, error) {
	discordToken, err := ExchangeCode(code, common.Config.Origin+authUrl)
	if err != nil {
		return "", err
	}

	discordUser, err := GetUser(discordToken.AccessToken)
	if err != nil {
		return "", err
	}
	token := modules.GenerateToken()

	user := &schemas.TwitterUser{
		TwitterID:    discordUser.ID,
		Token:        token,
		Username:     discordUser.Username + "#" + discordUser.Discriminator,
		//AvatarURL:    GetProfilePhotoURL(discordUser.ID, discordUser.Avatar),
		Type:         0,
		IpHash:       modules.CalculateHash(ip),
		RefreshToken: discordToken.RefreshToken,
	}
	if discordUser.Discriminator == "0" {
		user.Username = discordUser.Username
	}

	dbUser, err := GetDBUserViaDiscordID(discordUser.ID)

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

		_, err = database.DB.NewUpdate().Where("id = ?", dbUser.ID).Model(dbUser).Exec(context.Background())
		if err != nil {
			return "", err
		}

		return dbUser.Token, nil
	}

	_, err = database.DB.NewInsert().Model(user).Exec(context.Background())
	if err != nil {
		return "An Error occurred", err
	}

	SendLoggerWebhook(WebhookData{
		Username:  discordUser.Username + "#" + discordUser.Discriminator,
		AvatarURL: GetProfilePhotoURL(discordUser.ID, discordUser.Avatar),
		Content:   fmt.Sprintf("User <@%s> has been registered to ReviewDB from %s", discordUser.ID, clientmod),
	})

	return token, nil
}
*/
