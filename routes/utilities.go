package routes

import (
	"errors"
	"net/http"
	"server-go/database"
	"server-go/modules"
)

func Authorize(r *http.Request) (*database.URUser, error) {

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