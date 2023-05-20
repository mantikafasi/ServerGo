package modules

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/v3/state"

	"server-go/common"
	"server-go/database"

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

			component := BanTimeSelectComponent(action[1] + ":" + action[2])
			response.Data.Content = option.NewNullableString("Select ban duration")
			response.Data.Components = &component
			//UpdateWebhook(data.Message.ID, Response{Components: []WebhookComponent{component}})

		} else if action[0] == "delete_and_ban" {

			banDuration, _ := strconv.ParseInt(data.Data.Values[0], 10, 32)
			reviewid, err := strconv.ParseInt(action[2], 10, 32)
			review := database.UserReview{}

			if err == nil {
				review, _ = GetReview(int32(reviewid))
			}

			err = BanUser(action[1], common.Config.AdminToken, int32(banDuration), review)
			err2 := DeleteReview(int32(reviewid), common.Config.AdminToken)

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
			review := database.UserReview{}

			if err == nil {
				review, _ = GetReview(int32(reviewid))
			}

			err = BanUser(action[1], common.Config.AdminToken, int32(banDuration), review)
			if err == nil {
				response.Data.Content = option.NewNullableString(fmt.Sprintf("Successfully banned user %s for %d days", action[1], int32(banDuration)))
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

func ExchangeCode(code, redirectURL string) (*oauth2.Token, error) {
	conf := &oauth2.Config{
		Endpoint:     oauthEndpoint,
		Scopes:       []string{"identify"},
		RedirectURL:  redirectURL,
		ClientID:     common.Config.ClientId,
		ClientSecret: common.Config.ClientSecret,
	}

	token, err := conf.Exchange(context.Background(), code)

	if err != nil {
		return nil, err
	} else {
		return token, nil
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

type EmbedField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Embed struct {
	Title  string       `json:"title"`
	Fields []EmbedField `json:"fields"`
	Footer EmbedFooter  `json:"footer"`
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

type WebhookData struct {
	Content    string             `json:"content"`
	Username   string             `json:"username"`
	AvatarURL  string             `json:"avatar_url"`
	Embeds     []Embed            `json:"embeds"`
	Components []WebhookComponent `json:"components"`
}

func SendReportWebhook(data WebhookData) error {
	return SendWebhook(common.Config.ReportWebhook, data)
}

func SendLoggerWebhook(data WebhookData) error {
	return SendWebhook(common.Config.LoggerWebhook, data)
}

func SendAppealWebhook(data WebhookData) error {
	return SendWebhook(common.Config.AppealWebhook, data)
}

func SendWebhook(url string, data WebhookData) error {
	body, err := json.Marshal(data)
	var resp *http.Response

	resp, err = http.Post(url, "application/json", strings.NewReader(string(body)))
	_, err = io.ReadAll(resp.Body)
	return err
}

func GetProfilePhotoURL(userid string, avatar string) string {
	return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.%s", userid, avatar, common.Ternary(strings.HasPrefix(avatar, "a_"), "gif", "png"))
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

func GenerateToken() string {
	b := make([]byte, 64)

	if _, err := rand.Read(b); err != nil {
		return ""
	}
	encoder := base64.StdEncoding.WithPadding(base64.NoPadding)
	token := encoder.EncodeToString(b)

	return "rdb." + token
}
