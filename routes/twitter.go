package routes

import (
	"encoding/json"
	"io"
	"net/http"
	"server-go/common"
	"server-go/database/schemas"
	modules "server-go/modules/twitter"
	"strings"

	"github.com/go-chi/chi"
)

func ReviewDBTwitterAuth(w http.ResponseWriter, r *http.Request) {

	user, err := modules.AddTwitterUser(r.URL.Query().Get("code"), r.Header.Get("CF-Connecting-IP"))

	res := struct {
		schemas.TwitterUser
		Token string `json:"token"`
	}{
		TwitterUser: *user,
		Token:       user.Token,
	}

	if err != nil {
		println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response, _ := json.Marshal(res)
	io.WriteString(w, string(response))
}

func AddTwitterReview(w http.ResponseWriter, r *http.Request) {
	response := Response{}

	user, err := AuthorizeTwitter(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var data schemas.TwitterRequestData
	err = json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data.ProfileID = chi.URLParam(r, "profileid")

	if len(data.Comment) > 1000 {
		response.Message = "Comment Too Long"
	} else if len(strings.TrimSpace(data.Comment)) == 0 {
		response.Message = "Write Something Guh"
	}

	res, err := modules.AddReview(user, data)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte(res))
	w.WriteHeader(http.StatusOK)
}

type ReviewsResponseTwitter struct {
	Message     string                      `json:"message"` // for errors
	HasNextPage bool                        `json:"hasNextPage"`
	ReviewCount int                         `json:"reviewCount"`
	Reviews     []schemas.TwitterUserReview `json:"reviews"`
}

func GetTwitterReviews(w http.ResponseWriter, r *http.Request) {
	userid := chi.URLParam(r, "profileid")
	reviews, count, err := modules.GetTwitterReviews(userid, 0)

	res := ReviewsResponseTwitter{
		HasNextPage: len(reviews) > 50,
		ReviewCount: count,
	}

	if len(reviews) > 50 {
		res.Reviews = reviews[:len(reviews)-1]
	} else if len(reviews) == 0 {
		res.Reviews = []schemas.TwitterUserReview{}
		// we dont wanna send null
	} else {
		res.Reviews = reviews
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	common.SendStructResponse(w, res)
}

func HandleTwitterRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		AddTwitterReview(w, r)
	case "GET":
		GetTwitterReviews(w, r)
	}
}