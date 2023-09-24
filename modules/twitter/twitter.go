package modules

import (
	"context"
	"errors"
	"fmt"
	"server-go/common"
	"server-go/database"
	"server-go/database/schemas"
	"server-go/modules"

	discord_utils "server-go/modules/discord"

	"github.com/patrickmn/go-cache"
)

func GetReview(id int32) (*schemas.TwitterUserReview, error) {
	var review schemas.TwitterUserReview
	err := database.DB.NewSelect().Model(&review).Where("id = ?", id).Scan(context.Background())

	if err != nil {
		return nil, err
	}

	return &review, nil
}

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
		badges := GetBadgesOfUser(review.User.TwitterID)

		if review.User != nil {
			reviews[i].Sender.TwitterID = review.User.TwitterID
			reviews[i].Sender.AvatarURL = review.User.AvatarURL
			reviews[i].Sender.Username = review.User.Username
			reviews[i].Sender.DisplayName = review.User.DisplayName
			reviews[i].Sender.ID = review.User.ID
			reviews[i].Sender.Badges = badges
		}
		reviews[i].Timestamp = review.TimestampStr.Unix()
	}

	return reviews, count, nil
}

func GetBadgesOfUser(twitterID string) []schemas.TwitterUserBadge {
	userBadges := []schemas.TwitterUserBadge{}

	badges, _ := GetAllBadges()
	for _, badge := range badges {

		if badge.TargetTwitterID == twitterID {
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

	database.DB.NewSelect().Distinct().Model(&users).Column("twitter_id", "type").Where("type = ? or type = ?", 1, -1).Scan(context.Background(), &users)

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
		return nil, err
	}

	return &user, nil
}

func AddTwitterUser(code string, ip string) (*schemas.TwitterUser, error) {
	twitterToken, err := ExchangeCode(code)
	if err != nil {
		return nil, err
	}

	twitterUser, err := FetchUser(twitterToken.AccessToken)
	if err != nil {
		return nil, err
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
			return nil, errors.New("You have been banned from ReviewDB")
		}

		dbUser.Username = twitterUser.Data.Username
		dbUser.DisplayName = twitterUser.Data.Name
		dbUser.AvatarURL = twitterUser.Data.AvatarURL

		_, err = database.DB.NewUpdate().Where("id = ?", dbUser.ID).Model(dbUser).Exec(context.Background())
		if err != nil {
			return nil, err
		}

		return dbUser, nil
	}

	_, err = database.DB.NewInsert().Model(user).Exec(context.Background())
	if err != nil {
		println(err.Error())
		return nil, errors.New(common.ERROR)
	}

	discord_utils.SendLoggerWebhook(discord_utils.WebhookData{
		Username:  twitterUser.Data.Username,
		AvatarURL: twitterUser.Data.AvatarURL,
		Content:   fmt.Sprintf("User %s (%s) has been registered to ReviewDB Twitter", twitterUser.Data.Username, twitterUser.Data.ID),
	})
	return dbUser, nil
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

	review := &schemas.TwitterUserReview{
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

func DeleteReview(user *schemas.TwitterUser, reviewID int32) (err error) {
	review, err := GetReview(reviewID)
	if err != nil {
		fmt.Println(err.Error())
		return errors.New("Invalid Review ID")
	}

	if (review.ReviewerID == user.TwitterID) || user.IsAdmin() {
		//LogAction("DELETE", review, user.ID)

		_, err = database.DB.NewDelete().Model(review).Where("id = ?", reviewID).Exec(context.Background())
		if err != nil {
			println(err.Error())
			return errors.New(common.ERROR)
		}
		return nil
	}
	return errors.New("You are not allowed to delete this review")
}

func ReportReview(user *schemas.TwitterUser, reviewID int32) error {

	if user.IsBanned() {
		return errors.New("You cant report reviews while banned")
	}

	reportCount, _ := GetReportCountInLastHour(user.TwitterID)

	if reportCount > 20 {
		return errors.New("You are reporting too much")
	}

	count, _ := database.DB.NewSelect().Model(&schemas.TwitterReviewReport{}).Where("review_id = ? AND reporter_id = ?", reviewID, user.TwitterID).Count(context.Background())
	if count > 0 {
		return errors.New("You have already reported this review")
	}

	review, err := GetReview(reviewID)
	if err != nil {
		return errors.New("Invalid Review ID")
	}

	if review.ReviewerID == user.TwitterID {
		return errors.New("You cant report your own review")
	}

	reportedUser, _ := GetDBUserViaTwitterID(review.ReviewerID)

	report := schemas.TwitterReviewReport{
		ReviewID:   reviewID,
		ReporterID: user.TwitterID,
	}

	/*
		reviewedUsername := "?"
		if reviewedUser, err := ArikawaState.User(discord.UserID(review.ProfileID)); err == nil {
			reviewedUsername = reviewedUser.Tag()
		}

				Components: []modules.WebhookComponent{
				{
					Type: 1,
					Components: []modules.WebhookComponent{
						{
							Type:     2,
							Label:    "Delete Review",
							Style:    4,
							CustomID: fmt.Sprintf("delete_review:%d", reviewID),
							Emoji: modules.WebhookEmoji{
								Name: "🗑️",
							},
						},
						{
							Type:     2,
							Label:    "Ban User",
							Style:    4,
							CustomID: fmt.Sprintf("ban_select:%s:%d", reportedUser.DiscordID, reviewID), //string(reportedUser.DiscordID)
							Emoji: modules.WebhookEmoji{
								Name:     "banned",
								ID:       "590237837299941382",
								Animated: true,
							},
						},
						{
							Type:     2,
							Label:    "Delete Review and Ban User",
							Style:    4,
							CustomID: fmt.Sprintf("select_delete_and_ban:%d:%s", reviewID, string(reportedUser.DiscordID)),
							Emoji: modules.WebhookEmoji{
								Name:     "banned",
								ID:       "590237837299941382",
								Animated: true,
							},
						},
					},
				},
			},
	*/

	webhookData := discord_utils.WebhookData{
		Username: "ReviewDB Twitter",
		Content:  "Reported Review",

		Embeds: []discord_utils.Embed{
			{
				Fields: []discord_utils.EmbedField{
					{
						Name:  "**Review ID**",
						Value: fmt.Sprint(review.ID),
					},
					{
						Name:  "**Content**",
						Value: fmt.Sprint(review.Comment),
					},
					{
						Name:  "**Author**",
						Value: formatUser(reportedUser.Username, reportedUser.TwitterID),
					},
					{
						Name:  "**Reviewed User**",
						Value: review.ProfileID,
					},
					{
						Name:  "**Reporter**",
						Value: formatUser(user.Username, user.TwitterID),
					},
				},
			},
		},
	}

	/*
		if reportedUser.TwitterID != user.TwitterID {
			webhookData.Components[0].Components = append(webhookData.Components[0].Components, WebhookComponent{
				Type:     2,
				Label:    "Ban Reporter",
				Style:    4,
				CustomID: fmt.Sprintf("ban_select:" + user.DiscordID + ":" + "0"),
				Emoji: WebhookEmoji{
					Name:     "banned",
					ID:       "590237837299941382",
					Animated: true,
				},
			})
		}
	*/

	err = discord_utils.SendWebhook(common.Config.ReportWebhook, webhookData)

	if err != nil {
		println(err.Error())
	}

	database.DB.NewInsert().Model(&report).Exec(context.Background())
	return nil
}

func GetReportCountInLastHour(twitterUserID string) (int, error) {
	count, err := database.DB.
		NewSelect().Table("reviewdb_twitter.reports").
		Where("reporter_id = ? AND timestamp > now() - interval '1 hour'", twitterUserID).
		Count(context.Background())
	if err != nil {
		return 0, err
	}
	return count, nil
}
