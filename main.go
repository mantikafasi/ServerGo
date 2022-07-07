package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"server-go/common"
	"server-go/database"
	"server-go/modules"
)

func main() {
	database.InitDB()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//	fmt.Fprintf(w, "Hello, world!")
		io.WriteString(w, "Main Page does not exist")
	})
	http.HandleFunc("/vote", func(w http.ResponseWriter, r *http.Request) {

		var jason map[string]interface{}
		json.NewDecoder(r.Body).Decode(&jason)

		res := modules.VoteStupidity(strconv.Itoa(int(jason["discordid"].(float64))), jason["token"].(string), int32(jason["stupidity"].(float64)))

		io.WriteString(w, res)
	})

	http.HandleFunc("/getuser", func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.ParseInt(r.URL.Query().Get("discordid"), 10, 64)
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

		var jason map[string]interface{}
		json.NewDecoder(r.Body).Decode(&jason)

		res, err := modules.AddReview(strconv.Itoa(int(jason["userid"].(float64))), jason["token"].(string), jason["comment"].(string))
		fmt.Println(err)
		io.WriteString(w, res)
	})

	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		token, err := modules.AddStupidityDBUser(r.URL.Query().Get("code"))

		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/error", http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "receiveToken/"+token, http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/URauth", func(w http.ResponseWriter, r *http.Request) {
		token, err := modules.AddUserReviewsUser(r.URL.Query().Get("code"))

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

	http.HandleFunc("/receiveToken", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "You have successfully logged in! Your token is: "+r.URL.Query().Get("token")+"\n\n You can now close this window.")
	})

	err := http.ListenAndServe(":"+common.GetConfig().Port, nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")

	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
