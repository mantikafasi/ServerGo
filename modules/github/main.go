package github

import (
	"context"
	"encoding/json"
	"net/http"
	"server-go/common"

	"golang.org/x/oauth2"
)

func Authorize(code string) {


}

var oauthEndpoint = oauth2.Endpoint{
	AuthURL:   "https://github.com/login/oauth/access_token",
	TokenURL:  "https://github.com/login/oauth/access_token",
	AuthStyle: oauth2.AuthStyleInParams,
}

// todo: make one global exchange code function since all oauth2 providers use the same thing
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

type GithubUser struct {
	Login string `json:"login"`
	ID int `json:"id"`
	NodeID string `json:"node_id"`
	AvatarURL string `json:"avatar_url"`
}


func GetUserInfo(accessToken string) (user *GithubUser, err error) {
	httpClient := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Add("Authorization", "Bearer " + accessToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)

	if err != nil {
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&user)
	return
}