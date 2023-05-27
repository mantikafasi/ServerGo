package modules

import (
	"context"
	"server-go/common"

	"golang.org/x/oauth2"
)

var oauthEndpoint = oauth2.Endpoint{
	AuthURL:   common.Config.Twitter.ApiEndpoint + "/oauth2/authorize",
	TokenURL:  common.Config.Twitter.ApiEndpoint + "/oauth2/token",
	AuthStyle: oauth2.AuthStyleInParams,
}

func ExchangeCode(code string) (*oauth2.Token, error) {
	conf := &oauth2.Config{
		Endpoint:     oauthEndpoint,
		Scopes:       []string{"identify"},
		RedirectURL:  "https://manti.vendicated.dev/api/reviewdb-twitter/auth",
		ClientID:     common.Config.Twitter.ClientID,
		ClientSecret: common.Config.Twitter.ClientSecret,
	}

	token, err := conf.Exchange(context.Background(), code)

	if err != nil {
		return nil, err
	} else {
		return token, nil
	}
}

// TODO: sanitize config
func FetchUser(token string) {
	// TODO IMPLEMENT
}
