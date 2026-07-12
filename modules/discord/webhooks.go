package discord

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"server-go/common"
	"server-go/database/schemas"
	github_module "server-go/modules/github"
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
						Name:  "Reviewed Target",
						Value: reviewTargetValue(review),
					},
				},
			},
		},
	})
}

func reviewTargetValue(review *schemas.UserReview) string {
	if review.Type == schemas.ReviewTypeGithubRepository || schemas.IsGithubRepositoryProfileID(review.ProfileID) {
		repositoryID := schemas.GithubRepositoryIDFromProfileID(review.ProfileID)
		repository, err := github_module.GetRepositoryByID(repositoryID)
		if err == nil && repository.FullName != "" {
			if repository.HTMLURL != "" {
				return fmt.Sprintf("[%s](%s)\nGitHub Repository ID: %d", repository.FullName, repository.HTMLURL, repositoryID)
			}
			return fmt.Sprintf("%s\nGitHub Repository ID: %d", repository.FullName, repositoryID)
		}
		return fmt.Sprintf("GitHub Repository ID: %d", repositoryID)
	}

	reviewedUsername := "?"
	if reviewedUser, err := ArikawaState.User(discord.UserID(review.ProfileID)); err == nil {
		reviewedUsername = reviewedUser.Tag()
	}
	return common.FormatUser(reviewedUsername, 0, strconv.FormatInt(review.ProfileID, 10))
}

func SendReportWebhook(reporter *schemas.URUser, review *schemas.UserReview, reportedUser *schemas.URUser) error {

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
						Name:  "**Reviewed Target**",
						Value: reviewTargetValue(review),
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
			CustomID: "ban_select:" + reporter.DiscordID + ":0",
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
