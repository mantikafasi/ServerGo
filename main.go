package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"server-go/common"
	"server-go/database"
	"server-go/modules"
)

func main() {
	database.InitDB()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Main Page does not exist")
	})
	http.HandleFunc("/vote", func(w http.ResponseWriter, r *http.Request) {

		var data modules.SDB_RequestData
		json.NewDecoder(r.Body).Decode(&data)

		fmt.Println("/vote ", data.DiscordID," ",data.Stupidity)

		res := modules.VoteStupidity(data.DiscordID, data.Token, data.Stupidity)

		io.WriteString(w, res)
	})

	http.HandleFunc("/getuser", func(w http.ResponseWriter, r *http.Request) {

		userID, err := strconv.ParseInt(r.URL.Query().Get("discordid"), 10, 64)
		fmt.Println("/getuser ", userID)

		if err != nil {
			fmt.Println(err)
			io.WriteString(w, "An Error occurred\n")
			return
		}

		stupidity, error := modules.GetStupidity(userID)
		if error != nil {
			io.WriteString(w, "An Error occurred\n")
			return
		}
		if stupidity == -1 {
			io.WriteString(w, "None")
			return
		}
		io.WriteString(w, strconv.Itoa(stupidity))
	})

	http.HandleFunc("/getUserReviews", func(w http.ResponseWriter, r *http.Request) {
		
		userID, err := strconv.ParseInt(r.URL.Query().Get("discordid"), 10, 64)
		fmt.Println("/getUserReviews ", userID)
		if err != nil {
			fmt.Println(err)
			io.WriteString(w, "An Error occurred\n")
			return
		}

		reviews, err := modules.GetReviews(userID)
		if err != nil {
			fmt.Println(err)
			io.WriteString(w, "An Error occurred\n")
			return
		}
		if reviews == "null" {
			reviews = "[]"
		}
		io.WriteString(w, reviews)
	})

	http.HandleFunc("/addUserReview", func(w http.ResponseWriter, r *http.Request) {
		var data modules.UR_RequestData
		json.NewDecoder(r.Body).Decode(&data)

		fmt.Println("/addUserReview ", data.DiscordID," ",data.Comment)

		if len(data.Comment) > 1000 {
			io.WriteString(w, "Comment Too Long")
			return

		} else if len(strings.TrimSpace(data.Comment)) == 0 {
			io.WriteString(w, "Write Something Guh")
			return
		}

		res, err := modules.AddReview(data.DiscordID, data.Token, data.Comment,int32(data.ReviewType))
		fmt.Println(err)
		io.WriteString(w, res)
	})

	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		token, err := modules.AddStupidityDBUser(r.URL.Query().Get("code"))
		fmt.Println("/auth ")

		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/error", http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "receiveToken/"+token, http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/URauth", func(w http.ResponseWriter, r *http.Request) {
		token, err := modules.AddUserReviewsUser(r.URL.Query().Get("code"))
		fmt.Println("/URauth ")
		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/error", http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "receiveToken/"+token, http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "An Error occurred\n")
	})

	http.HandleFunc("/receiveToken/", func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.URL.Path, "/receiveToken/")
		io.WriteString(w, "You have successfully logged in! Your token is: "+token+"\n\n You can now close this window.")
	})

	err := http.ListenAndServe(":"+common.Config.Port, nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")

	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
