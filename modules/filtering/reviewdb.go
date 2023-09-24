package filtering

import (
	"errors"
	"server-go/common"
	"server-go/database/schemas"
	"server-go/modules"
	discord_utils "server-go/modules/discord"
)

type FilterFunction func(user *schemas.URUser, review *schemas.UserReview) error

var ReviewDB []FilterFunction

func init() {

	ReviewDB = []FilterFunction{

		func(reviewer *schemas.URUser, review *schemas.UserReview) (err error) {
			if !(review.Type == 0 || review.Type == 1) && reviewer.Type != 1 {
				err = errors.New(common.INVALID_REVIEW_TYPE)
			}
			return
		},

		func(reviewer *schemas.URUser, review *schemas.UserReview) (err error) {
			if reviewer.Type != 1 && discord_utils.ContainsCustomDiscordEmoji(review.Comment) {
				err = errors.New("You are not allowed to use custom emojis")
			}
			return
		},

		func(reviewer *schemas.URUser, review *schemas.UserReview) (err error) {
			if reviewer.Type != 1 && common.ContainsURL(review.Comment) {
				err = errors.New("You are not allowed to have URLs in your review")
			}
			return
		},

		func(reviewer *schemas.URUser, review *schemas.UserReview) (err error) {
			if reviewer.OptedOut {
				err = errors.New(common.OPTED_OUT)
			}
			return
		},

		func(reviewer *schemas.URUser, review *schemas.UserReview) (err error) {
			if reviewer.IsBanned() {
				err = errors.New("You have been banned from ReviewDB until " + reviewer.BanInfo.BanEndDate.Format("2006-01-02 15:04:05") + "UTC")
			}
			return
		},

		func(reviewer *schemas.URUser, review *schemas.UserReview) (err error) {
			if reviewer.Type == -1 {
				err = errors.New("You have been banned from ReviewDB permanently")
			}
			return
		},

		func(reviewer *schemas.URUser, review *schemas.UserReview) (err error) {
			count, _ := modules.GetReviewCountInLastHour(reviewer.ID)
			if count > 20 {
				err = errors.New("You are reviewing too much")
			}
			return
		},

		func(reviewer *schemas.URUser, review *schemas.UserReview) (err error) {
			if common.LightProfanityDetector.IsProfane(review.Comment) {
				err = errors.New("Your review contains profanity")
			}
			return
		},

		func(reviewer *schemas.URUser, review *schemas.UserReview) (err error) {
			if common.ProfanityDetector.IsProfane(review.Comment) {
				review.ID = -1
				modules.BanUser(reviewer.DiscordID, common.Config.AdminToken, 7, *review)
				discord_utils.SendUserBannedWebhook(reviewer, review)
				err = errors.New("Because of trying to post a profane review, you have been banned from ReviewDB for 1 week")
			}
			return
		},
	}
}
