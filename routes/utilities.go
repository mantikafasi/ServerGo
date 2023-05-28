package routes

import (
	"errors"
	"net/http"
	"server-go/database/schemas"
	"server-go/modules"
	 twitter_modules "server-go/modules/twitter"
)

func Authorize(r *http.Request) (*schemas.URUser, error) {

	var token = r.Header.Get("Authorization")

	if token == "" {
		return nil, errors.New("Unauthorized")
	}
	user, err := modules.GetDBUserViaToken(token)

	if err != nil {
		return nil, errors.New("Unauthorized")
	}

	return &user, nil
}

// maybe using a middleware would be better but it prevents me from adding metrics
func AuthorizeTwitter(r *http.Request) (*schemas.TwitterUser, error) {

	var token = r.Header.Get("Authorization")

	if token == "" {
		return nil, errors.New("Unauthorized")
	}

	user, err := twitter_modules.GetDBUserViaToken(token)
	if err != nil {
		return nil, errors.New("Unauthorized")
	}

	return user, nil
}
