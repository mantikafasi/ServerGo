package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"server-go/constantants"
	"server-go/modules"
	"strconv"
	"encoding/json"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func main() {

	DB := bun.NewDB(sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithAddr(constantants.DBIP),
		pgdriver.WithUser(constantants.DBUSER),
		pgdriver.WithPassword(constantants.DBPASSWORD),
		pgdriver.WithDatabase(constantants.DBNAME),
		pgdriver.WithTLSConfig(nil),
	)), pgdialect.New())


	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//	fmt.Fprintf(w, "Hello, world!")
		io.WriteString(w, "Main Page does not exist")

	})
	http.HandleFunc("/vote", func(w http.ResponseWriter, r *http.Request) {
		body,_ := ioutil.ReadAll(r.Body)
		//convert body to json
		
		var jason map[string]interface{}
		json.Unmarshal(body, &jason)
		
		res := modules.VoteStupidity(DB,int64(jason["discordid"].(float64)),jason["token"].(string),int32(jason["stupidity"].(float64)))
	
		io.WriteString(w, res)
	})

	http.HandleFunc("/getuser", func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.ParseInt(r.URL.Query().Get("discordid"), 10, 64)
		if err != nil {
			fmt.Println(err)
			io.WriteString(w, "An Error Occured\n")
			return
		}

		stupidity,error := modules.GetStupidity(DB, userID)
		if error != nil {
			io.WriteString(w, "An Error Occured\n")
			return
		}
		io.WriteString(w, strconv.Itoa(stupidity))
	})
	
	http.HandleFunc("/getUserReviews",func(w http.ResponseWriter, r *http.Request) {
		
		userID, err := strconv.ParseInt(r.URL.Query().Get("discordid"), 10, 64)
		if err != nil {
			fmt.Println(err)
			io.WriteString(w, "An Error Occured\n")
			return
		}

		reviews, err := modules.GetReviews(DB, userID)
		if err != nil {
			fmt.Println(err)
			io.WriteString(w, "An Error Occured\n")
			return
		}
		io.WriteString(w, reviews)
	})

	http.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		token,err := modules.AddStupidityDBUser(DB,r.URL.Query().Get("code"))
		
		if err != nil {
			fmt.Println(err)
			http.Redirect(w,r,"/error",http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w,r,"receiveToken?token="+token,http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/URauth", func(w http.ResponseWriter, r *http.Request) {
		token, err := modules.AddUserReviewsUser(DB,r.URL.Query().Get("code"))

		if err != nil {
			fmt.Println(err)
			http.Redirect(w,r,"/error",http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w,r,"receiveToken?token="+token,http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "An Error Occured\n")
	})

	http.HandleFunc("/receiveToken", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "You have successfully logged in! Your token is: "+r.URL.Query().Get("token")+"\n\n You can now close this window.")
	})



	err := http.ListenAndServe(":8080", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")

	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}