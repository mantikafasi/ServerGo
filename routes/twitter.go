package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"server-go/common"
	"server-go/database/schemas"
	"server-go/modules"
	modules_twitter "server-go/modules/twitter"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
)

func ReviewDBTwitterAuth(w http.ResponseWriter, r *http.Request) {

	user, err := modules_twitter.AddTwitterUser(r.URL.Query().Get("code"), r.Header.Get("CF-Connecting-IP"))

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
	text, _ := json.Marshal(res)

	response := fmt.Sprintf(`
	<div data-reviewdb-auth = '%s' />`, text)

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

	res, err := modules_twitter.AddReview(user, data)

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
	reviews, count, err := modules_twitter.GetTwitterReviews(userid, 0)

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

func DeleteReviewTwitter(w http.ResponseWriter, r *http.Request) {
	user, err := AuthorizeTwitter(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized, please reauthroize"))
		return
	}
	reviewID, err := strconv.Atoi(chi.URLParam(r, "profileid"))

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid review id"))
		return
	}

	err = modules_twitter.DeleteReview(user, int32(reviewID))
	if err != nil {
		w.Write([]byte("An error occured while deleting review"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Successfully deleted review"))
	w.WriteHeader(http.StatusOK)
}

func ReportTwitterReview(w http.ResponseWriter, r *http.Request) {
	user, err := AuthorizeTwitter(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized, please reauthroize"))
		return
	}

	var data modules.ReportData
	json.NewDecoder(r.Body).Decode(&data)

	if data.ReviewID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid Request"))
		return
	}

	err = modules_twitter.ReportReview(user, data.ReviewID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("An error occured while reporting review"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Successfully reported review"))
}

func HandleTwitterRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		AddTwitterReview(w, r)
	case "GET":
		GetTwitterReviews(w, r)
	case "DELETE":
		DeleteReviewTwitter(w, r)
	}
}
