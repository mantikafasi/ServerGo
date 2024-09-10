package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"server-go/common"
	"server-go/database/schemas"
	"server-go/modules"
	discord_utils "server-go/modules/discord"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
)

type InteractionsData struct {
	Type int `json:"type"` // 1 = ping
	Data struct {
		ID         string                           `json:"custom_id"`
		Values     []string                         `json:"values"`
		Components []discord_utils.WebhookComponent `json:"components"`
	}
	Message struct {
		Content string `json:"content"`
		ID      string `json:"id"`
	}

	Member struct {
		User struct {
			ID            string `json:"id"`
			Username      string `json:"username"`
			Discriminator string `json:"discriminator"`
		} `json:"user"`
	} `json:"member"`
}

func BanTimeSelectComponent(userid string) discord.ContainerComponents {
	return BanTimeSelectComponentWithID(userid, "ban_user")
}

func BanTimeSelectComponentWithID(userid string, componentID string) discord.ContainerComponents {
	return discord.ContainerComponents{
		&discord.ActionRowComponent{
			&discord.StringSelectComponent{
				CustomID:    discord.ComponentID(componentID + ":" + userid),
				Placeholder: "Select ban time",
				Options: []discord.SelectOption{
					{
						Label: "1 day",
						Value: "1",
					},
					{
						Label: "3 days",
						Value: "3",
					},
					{
						Label: "1 week",
						Value: "7",
					},
					{
						Label: "1 month",
						Value: "30",
					},
				},
			},
		},
	}
}

func AppealDenyTextComponent(appealID int32) discord.ContainerComponents {

	return discord.ContainerComponents{
		&discord.ActionRowComponent{
			&discord.TextInputComponent{
				Label:       "Deny Reason",
				Placeholder: "You wrote such a dumb reason even I could think of a better one",
				Style:       discord.TextInputParagraphStyle,
				Required:    true,
				Value:     "You wrote such a dumb reason even I could think of a better one"
			},
		},
	}
}

func Interactions(data InteractionsData) (string, error) {

	if data.Type == 1 {
		return "{\"type\":1}", nil //copilot I hope you die
	}
	response := api.InteractionResponse{}

	response.Type = 4

	response.Data = &api.InteractionResponseData{}

	userid, _ := strconv.ParseInt(data.Member.User.ID, 10, 64)

	action := strings.Split(data.Data.ID, ":")

	if (data.Type == 3 || data.Type == 5) && modules.IsUserAdminDC(userid) {

		response.Data.Embeds = &[]discord.Embed{{
			Footer: &discord.EmbedFooter{
				Text: fmt.Sprintf("Admin: %s#%s (%s)", data.Member.User.Username, data.Member.User.Discriminator, data.Member.User.ID),
			},
		}}

		firstVariable, _ := strconv.ParseInt(action[1], 10, 32) // if action is delete review or delete_and_ban its reviewid otherwise userid
		if action[0] == "delete_review" {
			err := modules.DeleteReview(int32(firstVariable), common.Config.AdminToken)
			if err == nil {
				response.Data.Content = option.NewNullableString("Successfully Deleted review with id " + action[1])
			} else {
				response.Data.Content = option.NewNullableString(err.Error())
			}
		} else if action[0] == "ban_select" {

			component := BanTimeSelectComponent(action[1] + ":" + action[2])
			response.Data.Content = option.NewNullableString("Select ban duration")
			response.Data.Components = &component

		} else if action[0] == "delete_and_ban" {

			banDuration, _ := strconv.ParseInt(data.Data.Values[0], 10, 32)
			reviewid, err := strconv.ParseInt(action[2], 10, 32)
			review := schemas.UserReview{}

			if err == nil {
				review, _ = modules.GetReview(int32(reviewid))
			}

			err = modules.BanUser(action[1], common.Config.AdminToken, int32(banDuration), review)
			err2 := modules.DeleteReview(int32(reviewid), common.Config.AdminToken)

			if err == nil && err2 == nil {
				response.Data.Content = option.NewNullableString(fmt.Sprintf("Successfully deleted review with id %s and banned user %s for %d days", action[2], action[1], int32(banDuration)))
			} else if err == nil && err2 != nil {
				response.Data.Content = option.NewNullableString(fmt.Sprintf("Successfully banned user %s for %d days and failed to delete review with id %s\n Reason: %s", action[1], int32(banDuration), action[2], err2.Error()))
				fmt.Println(err)
			} else if err != nil && err2 != nil {
				response.Data.Content = option.NewNullableString(fmt.Sprintf("Failed to delete review with id %s and failed to ban user %s for %d days\nBan Fail Reason: %s\nReview Delete fail reason:%s", action[2], action[1], int32(banDuration), err.Error(), err2.Error()))
				fmt.Println(err, err2)
			} else {
				response.Data.Content = option.NewNullableString(fmt.Sprintf("Failed to ban user with id %s and successfully deleted review with id %s\nReason: %s", action[1], action[2], err.Error()))
			}

			response.Type = 7 // update message

			response.Data.Components = &discord.ContainerComponents{} // remove components

		} else if action[0] == "select_delete_and_ban" {
			component := BanTimeSelectComponentWithID(action[2]+":"+action[1], "delete_and_ban")
			response.Data.Components = &component
			response.Data.Content = option.NewNullableString("Select ban duration & delete review")

		} else if action[0] == "ban_user" {

			banDuration, _ := strconv.ParseInt(data.Data.Values[0], 10, 32)
			reviewid, err := strconv.ParseInt(action[2], 10, 32)
			review := schemas.UserReview{}

			if err == nil {
				review, _ = modules.GetReview(int32(reviewid))
			}

			err = modules.BanUser(action[1], common.Config.AdminToken, int32(banDuration), review)
			if err == nil {
				response.Data.Content = option.NewNullableString(fmt.Sprintf("Successfully banned user %s for %d days", action[1], int32(banDuration)))
			} else {
				response.Data.Content = option.NewNullableString(err.Error())
			}
			response.Type = 7 // update message

			response.Data.Components = &discord.ContainerComponents{} // remove components

		} else if action[0] == "accept_appeal" {
			appeal, err := modules.GetAppeal(int32(firstVariable))
			if err != nil {
				response.Data.Content = option.NewNullableString(err.Error())
				return InteractionResponse(&response), nil
			}

			if appeal.ActionTaken {
				response.Data.Content = option.NewNullableString("Appeal action already taken")
				return InteractionResponse(&response), nil
			}
			err = modules.AcceptAppeal(&appeal, appeal.UserID)

			if err == nil {
				response.Data.Content = option.NewNullableString(fmt.Sprintf("Successfully unbanned user %d", appeal.UserID))
			} else {
				response.Data.Content = option.NewNullableString(err.Error())
			}

		} else if action[0] == "text_deny_appeal" {
			appealId := int32(firstVariable)
			component := AppealDenyTextComponent(appealId)
			response.Data.Components = &component
			response.Data.Title = option.NewNullableString("Enter Deny Reason")
			response.Data.CustomID = option.NewNullableString(fmt.Sprintf("deny_appeal:%d", appealId))
			response.Type = 9 // modal
		} else if action[0] == "deny_appeal" {
			appeal, err := modules.GetAppeal(int32(firstVariable))

			if err != nil {
				response.Data.Content = option.NewNullableString(err.Error())
				return InteractionResponse(&response), nil
			}
			if appeal.ActionTaken {
				response.Data.Content = option.NewNullableString("Appeal action already taken")
				return InteractionResponse(&response), nil
			}
			denyReason := data.Data.Components[0].Components[0].Value
			err = modules.DenyAppeal(&appeal, denyReason)
			if err != nil {
				response.Data.Content = option.NewNullableString(err.Error())
			} else {
				response.Data.Content = option.NewNullableString("Successfully denied appeal\n\n ```" + denyReason + "```")
			}
		}
	}

	if response.Data.Content != nil || response.Data.Title != nil {
		b, err := json.Marshal(response)
		if err != nil {
			return "", err
		}
		fmt.Println(string(b))

		return string(b), nil
	}
	return "", errors.New("invalid interaction")
}

func InteractionResponse(response *api.InteractionResponse) string {
	b, _ := json.Marshal(response)
	return string(b)
}
