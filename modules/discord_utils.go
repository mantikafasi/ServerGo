package modules

import (
	"context"
	"encoding/json"
	"net/http"

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
	AuthURL:   common.GetConfig().ApiEndpoint + "/oauth2/authorize",
	TokenURL:  common.GetConfig().ApiEndpoint + "/oauth2/token",
	AuthStyle: oauth2.AuthStyleInParams,
}

func ExchangeCodePlus(code, redirectURL string) (string, error) {
	conf := &oauth2.Config{
		Endpoint:     oauthEndpoint,
		Scopes:       []string{"identify"},
		RedirectURL:  redirectURL,
		ClientID:     common.GetConfig().ClientId,
		ClientSecret: common.GetConfig().ClientSecret,
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
	req, _ := http.NewRequest(http.MethodGet, common.GetConfig().ApiEndpoint+"/users/@me", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&user)
		resp.Body.Close()
	}
	json.NewDecoder(resp.Body).Decode(&user)
	return user, nil
}

func ExchangeCode(token string) (string, error) {
	return ExchangeCodePlus(token, common.GetConfig().RedirectUri)
}
