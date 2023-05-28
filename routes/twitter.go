package routes

import (
	"encoding/json"
	"io"
	"net/http"
	"server-go/common"
	"server-go/database/schemas"
	modules "server-go/modules/twitter"
	"strings"
)

func ReviewDBTwitterAuth(w http.ResponseWriter, r *http.Request) {

	token, err := modules.AddTwitterUser(r.URL.Query().Get("code"), r.Header.Get("CF-Connecting-IP"))

	if err != nil {
		println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res := ReviewDBAuthResponse{
		Token: token,
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
	Response
	HasNextPage bool                        `json:"hasNextPage"`
	ReviewCount int                         `json:"reviewCount"`
	Reviews     []schemas.TwitterUserReview `json:"reviews"`
}

func GetTwitterReviews(w http.ResponseWriter, r *http.Request) {
	userid := r.URL.Query().Get("userid")
	reviews, count, err := modules.GetTwitterReviews(userid, 0)

	res := ReviewsResponseTwitter{
		HasNextPage: len(reviews) > 50,
		ReviewCount: count,
		Reviews:     reviews[:len(reviews)-1],
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	common.SendStructResponse(w, res)
}