package discord

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"server-go/common"
	"server-go/database/schemas"
	"strconv"

	"github.com/diamondburned/arikawa/v3/discord"
)

func SendUserBannedWebhook(reviewer *schemas.URUser, review *schemas.UserReview) {
	SendLoggerWebhook(WebhookData{
		Username: "ReviewDB",
		Content:  "User <@" + reviewer.DiscordID + "> has been banned for 1 week for trying to post a profane review",
		Embeds: []discord.Embed{
			{
				Fields: []discord.EmbedField{
					{
						Name:  "Review Content",
						Value: review.Comment,
					},
					{
						Name:  "ReviewDB ID",
						Value: strconv.Itoa(int(reviewer.ID)),
					},
					{
						Name:  "Reviewed Profile",
						Value: "<@" + strconv.FormatInt(int64(review.ProfileID), 10) + ">",
					},
				},
			},
		},
	})
}

func SendReportWebhook(reporter *schemas.URUser, review *schemas.UserReview, reportedUser *schemas.URUser) error {

	reviewedUsername := "?"
	if reviewedUser, err := ArikawaState.User(discord.UserID(review.ProfileID)); err == nil {
		reviewedUsername = reviewedUser.Tag()
	}

	sourceLang := ""
	translatedContent := ""
	if res, err := http.Get("https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl=en&dt=t&dj=1&source=input&q=" + url.QueryEscape(review.Comment)); err == nil {
		var trans common.Translate
		if err = json.NewDecoder(res.Body).Decode(&trans); err == nil {
			if trans.Src != "en" && trans.Confidence > 0.3 {
				sourceLang = " (" + trans.Src + ")"
				translatedContent = ""
				for _, sentence := range trans.Sentences {
					translatedContent += sentence.Trans + "\n"
				}
			}
		}
	}

	webhookData := WebhookData{
		Username: "ReviewDB",
		Content:  "Reported Review",
		Components: []WebhookComponent{
			{
				Type: 1,
				Components: []WebhookComponent{
					{
						Type:     2,
						Label:    "Delete Review",
						Style:    4,
						CustomID: fmt.Sprintf("delete_review:%d", review.ID),
						Emoji: discord.ComponentEmoji{
							Name: "🗑️",
						},
					},
					{
						Type:     2,
						Label:    "Ban User",
						Style:    4,
						CustomID: fmt.Sprintf("ban_select:%s:%d", reportedUser.DiscordID, review.ID), //string(reportedUser.DiscordID)
						Emoji: discord.ComponentEmoji{
							Name:     "banned",
							ID:       590237837299941382,
							Animated: true,
						},
					},
					{
						Type:     2,
						Label:    "Delete Review and Ban User",
						Style:    4,
						CustomID: fmt.Sprintf("select_delete_and_ban:%d:%s", review.ID, string(reportedUser.DiscordID)),
						Emoji: discord.ComponentEmoji{
							Name:     "banned",
							ID:       590237837299941382,
							Animated: true,
						},
					},
				},
			},
		},
		Embeds: []discord.Embed{
			{
				Fields: []discord.EmbedField{
					{
						Name:  "**Review ID**",
						Value: fmt.Sprint(review.ID),
					},
					{
						Name:  "**Content**",
						Value: fmt.Sprint(review.Comment),
					},
					{
						Name:  "**Translated Content" + sourceLang + "**",
						Value: translatedContent,
					},
					{
						Name:  "**Author**",
						Value: common.FormatUser(reportedUser.Username, reportedUser.ID, reportedUser.DiscordID),
					},
					{
						Name:  "**Reviewed User**",
						Value: common.FormatUser(reviewedUsername, 0, strconv.FormatInt(review.ProfileID, 10)),
					},
					{
						Name:  "**Reporter**",
						Value: common.FormatUser(reporter.Username, reporter.ID, reporter.DiscordID),
					},
				},
			},
		},
	}

	if translatedContent == "" {
		embed := webhookData.Embeds[0]
		// remove translated content field if no translation
		fields := make([]discord.EmbedField, 0)
		fields = append(fields, embed.Fields[:2]...)
		webhookData.Embeds[0].Fields = append(fields, embed.Fields[3:]...)
	}

	if reportedUser.DiscordID != reporter.DiscordID {
		webhookData.Components[0].Components = append(webhookData.Components[0].Components, WebhookComponent{
			Type:     2,
			Label:    "Ban Reporter",
			Style:    4,
			CustomID: fmt.Sprintf("ban_select:" + reporter.DiscordID + ":" + "0"),
			Emoji: discord.ComponentEmoji{
				Name:     "banned",
				ID:       590237837299941382,
				Animated: true,
			},
		})
	}

	err := SendWebhook(common.Config.ReportWebhook, webhookData)

	return err
}

func SendAppealWebhook(appeal *schemas.ReviewDBAppeal, user *schemas.URUser) {
	SendWebhook(common.Config.AppealWebhook,
		WebhookData{
			Username: "ReviewDB Appeals",
			Embeds: []discord.Embed{
				{
					Title: "Appeal Form",
					Fields: []discord.EmbedField{
						{
							Name:  "User",
							Value: common.FormatUser(user.Username, user.ID, user.DiscordID),
						},
						{
							Name:  "Reason to appeal",
							Value: appeal.AppealText,
						},
						{
							Name:  "Review Content",
							Value: user.BanInfo.ReviewContent,
						},
					},
				},
			},
			Components: []WebhookComponent{
				{
					Type: 1,
					Components: []WebhookComponent{
						{
							Type:     2,
							Label:    "Accept",
							Style:    3,
							CustomID: fmt.Sprintf("accept_appeal:%d", appeal.ID),
							Emoji: discord.ComponentEmoji{
								Name: "✅",
							},
						},
						{
							Type:     2,
							Label:    "Deny",
							Style:    4,
							CustomID: fmt.Sprintf("text_deny_appeal:%d", appeal.ID),
							Emoji: discord.ComponentEmoji{
								Name: "❌",
							},
						},
					},
				},
			},
		})
}
