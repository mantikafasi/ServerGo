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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"golang.org/x/exp/slices"
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

var Counters = map[string]prometheus.Counter{}
var TotalRequestCounter = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "total_request",
	Help: "Total request count",
})

func (c *Cors) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {

	metric := strings.Replace(pattern, "/", "", -1)
	if metric == "" {
		metric = "root"
	}

	if _, exists := Counters[metric]; !exists {
		Counters[metric] = prometheus.NewCounter(prometheus.CounterOpts{
			Name: metric,
			Help: "Number of requests on " + pattern,
		})
		prometheus.MustRegister(Counters[metric])
	}

	c.handler.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		TotalRequestCounter.Inc()
		Counters[metric].Inc()
		handler(w, r)
	})
}

var URUserCounter = prometheus.NewCounterFunc(prometheus.CounterOpts{
	Name: "user_count",
	Help: "Count of user reviews users",
}, func() float64 {
	userCount, err := modules.GetURUserCount()

	if err != nil {
		return 0
	}

	return float64(userCount)
})

var ReviewCounter = prometheus.NewCounterFunc(prometheus.CounterOpts{
	Name: "review_count",
	Help: "Count of total user reviews",
}, func() float64 {
	count, err := modules.GetReviewCount()

	if err != nil {
		return 0
	}

	return float64(count)
})

func (c *Cors) Handle(pattern string, handler http.Handler) {
	c.handler.Handle(pattern, handler)
}

func main() {

	common.InitCache()
	database.InitDB()

	prometheus.MustRegister(ReviewCounter)
	prometheus.MustRegister(URUserCounter)
	prometheus.MustRegister(TotalRequestCounter)

	mux := &Cors{http.NewServeMux()}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "artgallery/index.html")
	})

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("artgallery/static"))))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("artgallery/assets"))))

	mux.HandleFunc("/interactions", func(w http.ResponseWriter, r *http.Request) {
		var body []byte

		r.Body.Read(body)
		signature := r.Header.Get("X-Signature-Ed25519")
		timestamp := r.Header.Get("X-Signature-Timestamp")

		message := append(body, []byte(timestamp)...)
		if !common.VerifySignature([]byte(signature), message) {
			w.WriteHeader(401)
			return
		}
		var data modules.InteractionsData

		json.Unmarshal(body, &data)
		response, err := modules.Interactions(data)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(response))

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

		if slices.Contains(common.OptedOut, uint64(userID)) {
			reviews := append([]database.UserReview{{
				SenderUsername:  "ReviewDB",
				ProfilePhoto:    "https://cdn.discordapp.com/avatars/287555395151593473/7cd9b7a57f803b74009137f8bb073941.webp?size=128",
				Comment:         "This user has opted out of ReviewDB. It means you cannot review this user.",
				ReviewType:      1,
				SenderDiscordID: "287555395151593473",
				SystemMessage:   true,
				Badges:          []database.UserBadge{},
			}})
			jsonReviews, _ := json.Marshal(reviews)

			io.WriteString(w, string(jsonReviews))
			return
		}

		reviews, err := modules.GetReviews(userID)

		for i, j := 0, len(reviews)-1; i < j; i, j = i+1, j-1 {
			reviews[i], reviews[j] = reviews[j], reviews[i]
		}

		if err != nil {
			io.WriteString(w, "An Error occurred\n")
			return
		}

		if r.Header.Get("User-Agent") == "Aliucord (https://github.com/Aliucord/Aliucord)" && r.URL.Query().Get("noAds") != "true" {
			reviews = append([]database.UserReview{{
				SenderUsername:  "ReviewDB",
				ProfilePhoto:    "https://cdn.discordapp.com/avatars/287555395151593473/7cd9b7a57f803b74009137f8bb073941.webp?size=128",
				Comment:         "If you like the plugins I make, please consider supporting me at: \nhttps://github.com/sponsors/mantikafasi\n You can disable this in settings",
				ReviewType:      1,
				SenderDiscordID: "287555395151593473",
				SystemMessage:   true,
			}}, reviews...)
		}

		jsonReviews, _ := json.Marshal(reviews)
		reviewsStr := string(jsonReviews)

		if reviewsStr == "null" {
			reviewsStr = "[]"
		}

		io.WriteString(w, reviewsStr)
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

		if slices.Contains(common.OptedOut, uint64(data.DiscordID)) {
			io.WriteString(w, "This user opted out")
			return
		}

		res, err := modules.AddReview(data.DiscordID, data.Token, data.Comment, int32(data.ReviewType))
		if err != nil {
			println(err.Error())
		}
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
		Token  string `json:"token"`
		Status int32  `json:"status"`
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
				Token:  token,
				Status: 0,
			}
			response, _ := json.Marshal(res)
			io.WriteString(w, string(response))
			return
		}

		if err != nil {
			http.Redirect(w, r, "/error", http.StatusTemporaryRedirect)
			return
		}

		http.Redirect(w, r, "receiveToken/"+token, http.StatusTemporaryRedirect)
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
		Message    string `json:"message"`
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

	mux.HandleFunc("/getLastReviewID", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("discordid")

		w.Write([]byte(strconv.Itoa(int(modules.GetLastReviewID(id)))))
	})

	mux.Handle("/metrics", promhttp.Handler())

	err := http.ListenAndServe(":"+common.Config.Port, mux)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")

	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
