package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"server-go/common"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/v3/state"
	"golang.org/x/oauth2"
)

type WebhookData struct {
	Content    string             `json:"content"`
	Username   string             `json:"username"`
	AvatarURL  string             `json:"avatar_url"`
	Embeds     []Embed            `json:"embeds"`
	Components []WebhookComponent `json:"components"`
}

type EmbedField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type EmbedFooter struct {
	Text         string `json:"text"`
	IconURL      string `json:"icon_url"`
	ProxyIconURL string `json:"proxy_icon_url"`
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
	Value      string             `json:"value"`
	CustomID   string             `json:"custom_id"`
	Emoji      WebhookEmoji       `json:"emoji,omitempty"`
	Options    []ComponentOption  `json:"options,omitempty"`
	Components []WebhookComponent `json:"components"`
}

type ComponentOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

var ArikawaState *state.State
var emojiRegex *regexp.Regexp = regexp.MustCompile(`(<a?)?:\w+:(\d{18,19}>)?`)

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
	AuthURL:   common.Config.Discord.ApiEndpoint + "/oauth2/authorize",
	TokenURL:  common.Config.Discord.ApiEndpoint + "/oauth2/token",
	AuthStyle: oauth2.AuthStyleInParams,
}

func ContainsCustomDiscordEmoji(s string) bool {
	return emojiRegex.MatchString(s)
}

func ExchangeCode(code, redirectURL string) (*oauth2.Token, error) {
	conf := &oauth2.Config{
		Endpoint:     oauthEndpoint,
		Scopes:       []string{"identify"},
		RedirectURL:  redirectURL,
		ClientID:     common.Config.Discord.ClientID,
		ClientSecret: common.Config.Discord.ClientSecret,
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
	req, _ := http.NewRequest(http.MethodGet, common.Config.Discord.ApiEndpoint+"/users/@me", nil)
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
	req, _ := http.NewRequest(http.MethodGet, common.Config.Discord.ApiEndpoint+"/users/"+fmt.Sprint(userid), nil)
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

func SendWebhook(url string, data WebhookData) error {
	body, err := json.Marshal(data)
	var resp *http.Response

	resp, err = http.Post(url, "application/json", strings.NewReader(string(body)))
	_, err = io.ReadAll(resp.Body)

	return err
}

func SendLoggerWebhook(data WebhookData) error {
	return SendWebhook(common.Config.LoggerWebhook, data)
}