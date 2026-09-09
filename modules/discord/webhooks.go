package discord

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"server-go/common"
	"server-go/database/schemas"
	"server-go/modules/moderation"
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

// SendRegistrationBurstWebhook alerts moderators about a cluster of newly registered accounts.
func SendRegistrationBurstWebhook(ipHash string, users []schemas.URUser) error {
	registeredUsers := ""
	for i, user := range users {
		if i == 10 {
			registeredUsers += "…and more"
			break
		}
		registeredUsers += fmt.Sprintf("<@%s> (`%s`)\n", user.DiscordID, user.Username)
	}

	return SendWebhook(common.Config.ReportWebhook, WebhookData{
		Username: "ReviewDB",
		Content:  "Suspicious registration burst detected",
		Embeds: []discord.Embed{{
			Title: "Multiple registrations from one IP hash",
			Fields: []discord.EmbedField{
				{Name: "Registrations", Value: strconv.Itoa(len(users)) + " in the last 10 minutes"},
				{Name: "Accounts", Value: registeredUsers},
				{Name: "IP hash", Value: "`" + ipHash + "`"},
			},
		}},
		Components: []WebhookComponent{{
			Type: 1,
			Components: []WebhookComponent{{
				Type:     2,
				Label:    "Review bulk ban",
				Style:    4,
				CustomID: "ip_bulk_review:" + ipHash,
				Emoji:    discord.ComponentEmoji{Name: "⚠️"},
			}},
		}},
	})
}

func SendReportWebhook(reporter *schemas.URUser, review *schemas.UserReview, reportedUser *schemas.URUser, reportID int32) error {

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

	var commentSuffix string

	// Use translated content if available and not in a supported language
	contentToModerate := review.Comment
	if translatedContent != "" && sourceLang != "en" {
		contentToModerate = translatedContent
	}

	moderationResult, err := moderation.ModerateContent(contentToModerate)
	if err == nil {
		if !moderationResult.Flagged && len(moderationResult.Scores) == 0 {
			commentSuffix = ""
		} else {
			name, score := moderation.GetHighestScore(moderationResult)
			commentSuffix = fmt.Sprintf(" (%s - %d%%)", name, int(score*100))
		}
	} else {
		println(err.Error())
		commentSuffix = fmt.Sprintf(" (Rating: Error)")
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
						Value: fmt.Sprint(review.Comment, commentSuffix),
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
					{
						Name:  "**Report ID**",
						Value: fmt.Sprint(reportID),
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
			CustomID: fmt.Sprintf("ban_select:%s:0", reporter.DiscordID),
			Emoji: discord.ComponentEmoji{
				Name:     "banned",
				ID:       590237837299941382,
				Animated: true,
			},
		})
	}

	if commentSuffix != "" {
		err = SendWebhook(common.Config.ReportWebhook, webhookData)
	} else {
		err = SendWebhook(common.Config.JunkReportWebhook, webhookData)
	}

	return err
}

type CannedAppealDenyReason struct {
	Label string
	Value string
	Text  string
}

var CannedAppealDenyReasons = []CannedAppealDenyReason{
	{
		Label: "Idiot",
		Value: "idiot",
		Text:  "You wrote such a dumb reason even I could think of a better one",
	},
	{
		Label: "NSFW allegations",
		Value: "nsfw_allegations",
		Text:  "We don't allow allegations of this nature because they are inflammatory, difficult to verify, and tend to lead to unnecessary drama. This isn't a determination of whether the claim is true or false; it's simply content we don't want on ReviewDB. If you have credible evidence of actual misconduct, please report it to Discord or, where applicable, the relevant authorities rather than using ReviewDB to make or debate the allegation.",
	},
	{
		Label: "Hacked",
		Value: "hacked",
		Text:  "You are responsible for what happens to your account. In the future, ensure that only you has access to it",
	},
}

func GetCannedAppealDenyReason(value string) (string, bool) {
	for _, reason := range CannedAppealDenyReasons {
		if reason.Value == value {
			return reason.Text, true
		}
	}
	return "", false
}

func SendAppealWebhook(appeal *schemas.ReviewDBAppeal, user *schemas.URUser) {
	options := make([]discord.SelectOption, len(CannedAppealDenyReasons))
	for i, canned := range CannedAppealDenyReasons {
		options[i] = discord.SelectOption{
			Label: canned.Label,
			Value: canned.Value,
		}
	}

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
				{
					Type: 1,
					Components: []WebhookComponent{
						{
							Type:        3,
							CustomID:    fmt.Sprintf("canned_deny_appeal:%d", appeal.ID),
							Placeholder: "Deny with canned response...",
							Options:     options,
						},
					},
				},
			},
		})
}
