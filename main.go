package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"server-go/modules"
	"github.com/uptrace/bun"
	"database/sql"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

)

func main() {

	DB := bun.NewDB(sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithAddr("192.168.1.25:5432"),
		pgdriver.WithUser("manti"),
		pgdriver.WithPassword("mantikafasi3900plus"),
		pgdriver.WithDatabase("manti"),
		pgdriver.WithTLSConfig(nil),
	)), pgdialect.New())


	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//	fmt.Fprintf(w, "Hello, world!")
		io.WriteString(w, "Fart!\n")

	})
	http.HandleFunc("/vote", func(w http.ResponseWriter, r *http.Request) {
		
	})

	http.HandleFunc("/getUser", func(w http.ResponseWriter, r *http.Request) {
		stupidity,error := modules.GetStupidity(DB, 287555395151593473)
		if error != nil {
			io.WriteString(w, "An Error Occured\n")
			return
		}
		fmt.Printf("%v",stupidity)
		io.WriteString(w, "Stupidity: "+strconv.Itoa(stupidity)+"\n")

	})

	http.HandleFunc("/getReviews",func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("getReviews")
		
		userID, err := strconv.ParseInt(r.URL.Query().Get("discordid"), 10, 64)
		if err != nil {
			fmt.Println(err)
			return
		}
		reviews, err := modules.GetReviews(DB, userID)
		if err != nil {
			fmt.Println(err)
			return
		}
		io.WriteString(w, fmt.Sprintf("%v\n", reviews))
		fmt.Println(reviews)
	})

	err := http.ListenAndServe(":8080", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")

	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}