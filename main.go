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

type Cors struct {
    handler *http.ServeMux
}

func (c *Cors) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

    c.handler.ServeHTTP(w, r)
}

func (c *Cors) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	c.handler.HandleFunc(pattern, handler)
}


func main() {
	common.InitCache()
	database.InitDB()


	mux := &Cors{http.NewServeMux()}
	
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Main Page does not exist")
	})

	mux.HandleFunc("/vote", func(w http.ResponseWriter, r *http.Request) {

		var data modules.SDB_RequestData
		json.NewDecoder(r.Body).Decode(&data)


		res := modules.VoteStupidity(data.DiscordID, data.Token, data.Stupidity)

		io.WriteString(w, res)
	})

	mux.HandleFunc("/getuser", func(w http.ResponseWriter, r *http.Request) {

		userID, err := strconv.ParseInt(r.URL.Query().Get("discordid"), 10, 64)

		if err != nil {
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

	mux.HandleFunc("/getUserReviews", func(w http.ResponseWriter, r *http.Request) {

		userID, err := strconv.ParseInt(r.URL.Query().Get("discordid"), 10, 64)
		if err != nil {
			io.WriteString(w, "An Error occurred\n")
			return
		}

		reviews, err := modules.GetReviews(userID)
		if err != nil {
			io.WriteString(w, "An Error occurred\n")
			return
		}
		if reviews == "null" {
			reviews = "[]"
		}
		io.WriteString(w, reviews)
	})

	mux.HandleFunc("/addUserReview", func(w http.ResponseWriter, r *http.Request) {
		var data modules.UR_RequestData
		json.NewDecoder(r.Body).Decode(&data)


		if len(data.Comment) > 1000 {
			io.WriteString(w, "Comment Too Long")
			return

		} else if len(strings.TrimSpace(data.Comment)) == 0 {
			io.WriteString(w, "Write Something Guh")
			return
		}

		res, _ := modules.AddReview(data.DiscordID, data.Token, data.Comment, int32(data.ReviewType))
		io.WriteString(w, res)
	})

	mux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		token, err := modules.AddStupidityDBUser(r.URL.Query().Get("code"))

		if err != nil {
			http.Redirect(w, r, "/error", http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "receiveToken/"+token, http.StatusTemporaryRedirect)
	})

	type UR_AuthResponse struct {
		Token string `json:"token"`
		Status int32 `json:"status"`
	}

	mux.HandleFunc("/URauth", func(w http.ResponseWriter, r *http.Request) {
		clientmod := r.URL.Query().Get("clientMod")
		if clientmod == "" {
			clientmod = "aliucord"
		}

		token, err := modules.AddUserReviewsUser(r.URL.Query().Get("code"), clientmod)


		if r.URL.Query().Get("returnType") == "json" {
			if err != nil {
				io.WriteString(w, `{"token": "", "status": 1}`)
				return
			}

			res := UR_AuthResponse{
				Token: token,
				Status: 0,
			}
			response , _ := json.Marshal(res)
			io.WriteString(w, string(response))
			return
		}
		
		if err != nil {
			http.Redirect(w, r, "/error", http.StatusTemporaryRedirect)
			return
		}

		http.Redirect(w, r, "receiveToken/" + token, http.StatusTemporaryRedirect)
	})

	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "An Error occurred\n")
	})

	mux.HandleFunc("/reportReview", func(w http.ResponseWriter, r *http.Request) {
		var data modules.ReportData
		json.NewDecoder(r.Body).Decode(&data)

		if data.Token == "" || data.ReviewID == 0 {
			io.WriteString(w, "Invalid Request")
			return
		}
		err := modules.ReportReview(data.ReviewID, data.Token)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		io.WriteString(w, "Successfully Reported Review")
	})

	mux.HandleFunc("/receiveToken/", func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.URL.Path, "/receiveToken/")
		io.WriteString(w, "You have successfully logged in! Your token is: "+token+"\n\n You can now close this window.")
	})

	type Response struct {
		Successful bool   `json:"successful"`
		Message string `json:"message"`
	}

	mux.HandleFunc("/deleteReview", func(w http.ResponseWriter, r *http.Request) {
		var data modules.ReportData //both reportdata and deletedata are same
		json.NewDecoder(r.Body).Decode(&data)

		responseData := Response{
			Successful: false,
			Message:    "",
		}

		if data.Token == "" || data.ReviewID == 0 {
			responseData.Message = "Invalid Request"
			res, _ := json.Marshal(responseData)

			w.Write(res)
			return
		}

		err := modules.DeleteReview(data.ReviewID, data.Token)
		if err != nil {
			responseData.Message = err.Error()
			res, _ := json.Marshal(responseData)
			w.Write(res)
			return
		}
		responseData.Successful = true
		responseData.Message = "Successfully Deleted Review"
		res, _ := json.Marshal(responseData)
		w.Write(res)
	})
	
	

	err := http.ListenAndServe(":" + common.Config.Port, mux)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")

	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
