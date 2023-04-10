package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"server-go/common"
	"server-go/database"
	"server-go/legacy_routes"
	"server-go/modules"
	"server-go/routes"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Mux struct {
	*chi.Mux
}

var Counters = map[string]prometheus.Counter{}
var TotalRequestCounter = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "total_request",
	Help: "Total request count",
})
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

func (c *Mux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {

	metric := strings.NewReplacer("{", "", "}", "", "/", "", "*", "").Replace(pattern)

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
	Counters[metric].Inc()

	c.Handle(pattern, http.HandlerFunc(handler))
}

func cors(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		handler.ServeHTTP(w, r)
	})
}

func main() {

	common.InitCache()
	database.InitDB()

	prometheus.MustRegister(ReviewCounter)
	prometheus.MustRegister(URUserCounter)
	prometheus.MustRegister(TotalRequestCounter)

	mux := Mux{chi.NewRouter()}

	mux.Use(cors)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "artgallery/index.html")
	})

	mux.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("artgallery/static"))))

	mux.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("artgallery/assets"))))

	mux.HandleFunc("/interactions", routes.HandleInteractions)

	//StupidityDB

	mux.HandleFunc("/vote", routes.VoteStupidity)

	mux.HandleFunc("/getuser", routes.GetStupidity)

	mux.HandleFunc("/auth", routes.StupidityDBAuth)

	//ReviewDB

	mux.HandleFunc("/api/reviewdb/users", routes.GetUserInfo)

	mux.HandleFunc("/getUserReviews", legacy_routes.GetReviews)

	mux.HandleFunc("/addUserReview", legacy_routes.AddUserReview)

	mux.HandleFunc("/admins", routes.Admins)

	mux.HandleFunc("/URauth", legacy_routes.ReviewDBAuth)

	mux.HandleFunc("/api/reviewdb/auth", routes.ReviewDBAuth)

	mux.HandleFunc("/api/reviewdb/report", routes.ReportReview)

	mux.Route("/api/reviewdb/users/{discordid}/reviews", func(r chi.Router) {
		r.Get("/", routes.GetReviews)
		r.Post("/", routes.AddUserReview)
		r.Delete("/", routes.DeleteReview)
	})

	mux.HandleFunc("/api/reviewdb/badges", routes.GetAllBadges)

	mux.HandleFunc("/api/reviewdb/reports", routes.ReportReview)

	mux.HandleFunc("/api/reviewdb/reviews", routes.SearchReview)

	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "An Error occurred\n")
	})

	mux.HandleFunc("/api/reviewdb/", routes.HandleReviews)

	mux.HandleFunc("/reportReview", legacy_routes.ReportReview)

	mux.HandleFunc("/receiveToken/", routes.ReceiveToken)

	mux.HandleFunc("/deleteReview", legacy_routes.DeleteReview)

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
