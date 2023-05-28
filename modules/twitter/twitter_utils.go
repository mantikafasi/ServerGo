package modules

import (
	"context"
	"encoding/json"
	"net/http"
	"server-go/common"

	"golang.org/x/oauth2"
)

type TwitterUser struct {
	Data struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
		AvatarURL string `json:"profile_image_url"`
	}
}

var oauthEndpoint = oauth2.Endpoint{
	AuthURL:   common.Config.Twitter.ApiEndpoint + "/oauth2/authorize",
	TokenURL:  common.Config.Twitter.ApiEndpoint + "/oauth2/token",
	AuthStyle: oauth2.AuthStyleInParams,
}

func ExchangeCode(code string) (*oauth2.Token, error) {
	conf := &oauth2.Config{
		Endpoint:     oauthEndpoint,
		RedirectURL:  "https://manti.vendicated.dev/api/reviewdb-twitter/auth",
		ClientID:     common.Config.Twitter.ClientID,
		ClientSecret: common.Config.Twitter.ClientSecret,
	}

	token, err := conf.Exchange(context.Background(), code, oauth2.SetAuthURLParam("grant_type", "authorization_code"), oauth2.SetAuthURLParam("code_verifier", "challenge"))

	if err != nil {
		return nil, err
	} else {
		return token, nil
	}
}

func FetchUser(token string) (user *TwitterUser, err error) {
	req, _ := http.NewRequest(http.MethodGet, common.Config.Twitter.ApiEndpoint+"/users/me", nil)
	req.URL.Query().Add("user.fields", "id,name,username,profile_image_url")
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&user)
		resp.Body.Close()
		return user, nil
	}
	return nil, err
}
