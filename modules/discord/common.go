package discord

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"server-go/common"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
	"golang.org/x/oauth2"
)

type WebhookData struct {
	Content    string             `json:"content"`
	Username   string             `json:"username"`
	AvatarURL  string             `json:"avatar_url"`
	Embeds     []discord.Embed    `json:"embeds"`
	Components []WebhookComponent `json:"components"`
}

type WebhookComponent struct {
	Type       int                     `json:"type"`
	Style      int                     `json:"style"`
	Label      string                  `json:"label"`
	Value      string                  `json:"value"`
	CustomID   string                  `json:"custom_id"`
	Emoji      discord.ComponentEmoji  `json:"emoji,omitempty"`
	Options    []discord.CommandOption `json:"options,omitempty"`
	Components []WebhookComponent      `json:"components"`
}

var ArikawaState *state.State
var emojiRegex *regexp.Regexp = regexp.MustCompile(`(<a?)?:\w+:(\d{18,19}>)?`)

func init() {
	ArikawaState = state.New("Bot " + common.Config.BotToken)
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

func GetUser(token string) (user *discord.User, err error) {
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

func RefreshToken(token string) (*oauth2.Token, error) {
	conf := &oauth2.Config{
		Endpoint:     oauthEndpoint,
		Scopes:       []string{"identify"},
		ClientID:     common.Config.Discord.ClientID,
		ClientSecret: common.Config.Discord.ClientSecret,
	}

	tok := &oauth2.Token{
		RefreshToken: token,
	}

	newToken, err := conf.TokenSource(context.Background(), tok).Token()

	if err != nil {
		return nil, err
	} else {
		return newToken, nil
	}
}
