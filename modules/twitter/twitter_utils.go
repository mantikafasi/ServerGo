package modules

import (
	"context"
	"encoding/json"
	"net/http"
	"server-go/common"
	"server-go/database"
	"server-go/database/schemas"
	"time"
	"fmt"

	"golang.org/x/oauth2"
)

type TwitterUser struct {
	Data struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
		AvatarURL string `json:"profile_image_url"`
	} `json:"data"`
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
	req, _ := http.NewRequest(http.MethodGet, common.Config.Twitter.ApiEndpoint+"/users/me?user.fields=id%2Cname%2Cusername%2Cprofile_image_url", nil)
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&user)
		resp.Body.Close()
		return user, nil
	}
	return nil, err
}

func GetDBUserViaID(id string) (user schemas.TwitterUser, err error) {
	user = schemas.TwitterUser{}
	err = database.DB.NewSelect().Model(&user).Where("id = ?", id).Relation("BanInfo").Scan(context.Background(), &user)
	if user.BanInfo != nil && user.BanInfo.BanEndDate.Before(time.Now()) {
		user.BanInfo = nil
	}
	return
}

func formatUser(username string, twitterId string) string {
	return fmt.Sprintf("https://twitter.com/%s (%s)", username, twitterId)
}
