package routes

import (
	"errors"
	"net/http"
	"server-go/common"
	"server-go/database/schemas"
	"server-go/modules"
	twitter_modules "server-go/modules/twitter"
	"strconv"
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

func Notifications(w http.ResponseWriter, r *http.Request) {
	user, err := Authorize(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if r.Method == "PATCH" {
		notificationId, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = modules.ReadNotification(user, int32(notificationId))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}
}

func Error(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)

	response := Response{}
	response.Success = false
	response.Message = err.Error()
	common.SendStructResponse(w, response)
	return
}
