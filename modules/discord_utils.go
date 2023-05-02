package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/diamondburned/arikawa/v3/state"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"server-go/common"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"golang.org/x/oauth2"
)

var ArikawaState *state.State

func init() {
	ArikawaState = state.New("Bot " + common.Config.BotToken)
}

type DiscordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
}

var oauthEndpoint = oauth2.Endpoint{
	AuthURL:   common.Config.ApiEndpoint + "/oauth2/authorize",
	TokenURL:  common.Config.ApiEndpoint + "/oauth2/token",
	AuthStyle: oauth2.AuthStyleInParams,
}

type EmbedFooter struct {
	Text         string `json:"text"`
	IconURL      string `json:"icon_url"`
	ProxyIconURL string `json:"proxy_icon_url"`
}

type InteractionsData struct {
	Type int `json:"type"` // 1 = ping
	Data struct {
		ID     string   `json:"custom_id"`
		Values []string `json:"values"`
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
	return discord.ContainerComponents{
		&discord.ActionRowComponent{
			&discord.StringSelectComponent{
				CustomID:    discord.ComponentID("ban_user:" + userid),
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

func Interactions(data InteractionsData) (string, error) {

	if data.Type == 1 {
		return "{\"type\":1}", nil //copilot I hope you die
	}

	response := api.InteractionResponse{}

	response.Type = 4

	response.Data = &api.InteractionResponseData{}

	userid, _ := strconv.ParseInt(data.Member.User.ID, 10, 64)

	action := strings.Split(data.Data.ID, ":")

	if data.Type == 3 && IsUserAdminDC(userid) {

		response.Data.Embeds = &[]discord.Embed{{
			Footer: &discord.EmbedFooter{
				Text: fmt.Sprintf("Admin: %s#%s (%s)", data.Member.User.Username, data.Member.User.Discriminator, data.Member.User.ID),
			},
		}}

		firstVariable, _ := strconv.ParseInt(action[1], 10, 32) // if action is delete review or delete_and_ban its reviewid otherwise userid
		if action[0] == "delete_review" {
			err := DeleteReview(int32(firstVariable), common.Config.AdminToken)
			if err == nil {
				response.Data.Content = option.NewNullableString("Successfully Deleted review with id " + action[1])
			} else {
				response.Data.Content = option.NewNullableString(err.Error())
			}
		} else if action[0] == "ban_select" {

			component := BanTimeSelectComponent(action[1])
			response.Data.Content = option.NewNullableString("Select ban duration")
			response.Data.Components = &component
			//UpdateWebhook(data.Message.ID, Response{Components: []WebhookComponent{component}})

		} else if action[0] == "delete_and_ban" {
			component := BanTimeSelectComponent(action[2] + ":" + action[1])
			response.Data.Components = &component

			err := DeleteReview(int32(firstVariable), common.Config.AdminToken)
			if err == nil {
				response.Data.Content = option.NewNullableString("Successfully Deleted review with id " + action[1] + "\nSelect ban duration")
			} else {
				response.Data.Content = option.NewNullableString(err.Error()) // I hope this doesnt create error
			}
		} else if action[0] == "ban_user" {

			banDuration, _ := strconv.ParseInt(data.Data.Values[0], 10, 32)

			err := BanUser(action[1], common.Config.AdminToken, int32(banDuration))
			if err == nil {
				if len(action) == 3 {
					response.Data.Content = option.NewNullableString(fmt.Sprintf("Successfully banned user %s for %d days and deleted review with id %s", action[1], int32(banDuration), action[2]))
				} else {
					response.Data.Content = option.NewNullableString(fmt.Sprintf("Successfully banned user %s for %d days", action[1], int32(banDuration)))
				}
			} else {
				response.Data.Content = option.NewNullableString(err.Error())
			}
			response.Type = 7 // update message

			response.Data.Components = &discord.ContainerComponents{} // remove components
		}
	}
	if response.Data.Content.Val != "" {
		b, err := json.Marshal(response)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return "", errors.New("invalid interaction")
}

func UpdateWebhook(messageId string, payload interface{}) {
	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPatch, common.Config.DiscordWebhook+"/messages/"+messageId, strings.NewReader(string(jsonPayload)))
	req.Header.Set("Content-Type", "application/json")
	http.DefaultClient.Do(req)
}

func ExchangeCodePlus(code, redirectURL string) (string, error) {
	conf := &oauth2.Config{
		Endpoint:     oauthEndpoint,
		Scopes:       []string{"identify"},
		RedirectURL:  redirectURL,
		ClientID:     common.Config.ClientId,
		ClientSecret: common.Config.ClientSecret,
	}

	token, err := conf.Exchange(context.Background(), code)
	if err != nil {
		return "", err
	} else {
		return token.AccessToken, nil
	}

}

func GetUser(token string) (user *DiscordUser, err error) {
	// TODO discordid is always 0 fix
	req, _ := http.NewRequest(http.MethodGet, common.Config.ApiEndpoint+"/users/@me", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&user)
		resp.Body.Close()
		return user, nil
	}
	return nil, err
}

func GetUserViaID(userid int64) (user *DiscordUser, err error) {
	req, _ := http.NewRequest(http.MethodGet, common.Config.ApiEndpoint+"/users/"+fmt.Sprint(userid), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+common.Config.BotToken)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&user)
		resp.Body.Close()
		return user, nil
	}
	return nil, err
}

func ExchangeCode(token string) (string, error) {
	return ExchangeCodePlus(token, common.Config.RedirectUri)
}

type ReportWebhookEmbedField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Embed struct {
	Fields []ReportWebhookEmbedField `json:"fields"`
	Footer EmbedFooter               `json:"footer"`
}

type WebhookEmoji struct {
	Name     string `json:"name,omitempty"`
	ID       string `json:"id,omitempty"`
	Animated bool   `json:"animated,omitempty"`
}

type WebhookComponent struct {
	Type       int                `json:"type"`
	Style      int                `json:"style"`
	Label      string             `json:"label"`
	CustomID   string             `json:"custom_id"`
	Emoji      WebhookEmoji       `json:"emoji,omitempty"`
	Options    []ComponentOption  `json:"options,omitempty"`
	Components []WebhookComponent `json:"components"`
}

type ComponentOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type ReportWebhookData struct {
	Content    string             `json:"content"`
	Username   string             `json:"username"`
	AvatarURL  string             `json:"avatar_url"`
	Embeds     []Embed            `json:"embeds"`
	Components []WebhookComponent `json:"components"`
}

func SendReportWebhook(data ReportWebhookData) error {
	body, err := json.Marshal(data)
	var resp *http.Response

	resp, err = http.Post(common.Config.DiscordWebhook, "application/json", strings.NewReader(string(body)))
	bodyBytes, err := io.ReadAll(resp.Body)
	print(string(bodyBytes))
	return err
}

func GetProfilePhotoURL(userid string, avatar string) string {
	if strings.HasPrefix(avatar, "a_") {
		return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.gif", userid, avatar)
	} else {
		return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", userid, avatar)
	}
}

type Snowflake uint64

func (s *Snowflake) UnmarshalJSON(v []byte) error {
	parsed, err := strconv.ParseUint(strings.Trim(string(v), `"`), 10, 64)
	if err != nil {
		return err
	}

	*s = Snowflake(parsed)
	return nil
}
