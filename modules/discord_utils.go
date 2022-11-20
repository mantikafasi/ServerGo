package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"server-go/common"

	"golang.org/x/oauth2"
)

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
		return user,nil
	}
	return nil, err
}

func GetUserViaID(userid int64) (user *DiscordUser,err error) {
	req, _ := http.NewRequest(http.MethodGet, common.Config.ApiEndpoint+"/users/" + fmt.Sprint(userid), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot " + common.Config.BotToken)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&user)
		resp.Body.Close()
		return user,nil
	}
	return nil, err
}

func ExchangeCode(token string) (string, error) {
	return ExchangeCodePlus(token, common.Config.RedirectUri)
}

type ReportWebhookEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
}

type ReportWebhookEmbed struct {
	Fields []ReportWebhookEmbedField `json:"fields"`
}
type ReportWebhookData struct {
	Content string `json:"content"`
	Username string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	Embeds []ReportWebhookEmbed `json:"embeds"`

}

func SendReportWebhook(data ReportWebhookData) error {
	body,err := json.Marshal(data)

	_, err = http.Post(common.Config.DiscordWebhook, "application/json",strings.newReader(body))
	return err
}