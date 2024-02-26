package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"server-go/common"
	"server-go/database"
	"server-go/modules"
	"server-go/modules/discord"
	"server-go/routes"

	chiprometheus "server-go/middlewares/prometheus"

	"github.com/go-chi/chi"
	"github.com/go-chi/httprate"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	common.InitCache()
	database.InitDB()

	optedOutUsers, err := modules.GetOptedOutUsers()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	common.OptedOut = append(common.OptedOut, optedOutUsers...)

	mux := chi.NewRouter()
	prometheusMiddleware := chiprometheus.NewPatternMiddleware("reviewdb")

	mux.Use(routes.CorsMiddleware)
	mux.Use(httprate.LimitByRealIP(4, 1*time.Second))
	mux.Use(prometheusMiddleware)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "artgallery/index.html")
	})

	mux.Handle("/metrics", promhttp.Handler())

	mux.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("artgallery/static"))))

	mux.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("artgallery/assets"))))

	mux.HandleFunc("/interactions", routes.HandleInteractions)

	mux.HandleFunc("/receiveToken/{token}", routes.ReceiveToken)

	//StupidityDB

	mux.HandleFunc("/vote", routes.VoteStupidity)

	mux.HandleFunc("/getuser", routes.GetStupidity)

	mux.HandleFunc("/auth", routes.StupidityDBAuth)

	//ReviewDB

	mux.Route("/api/reviewdb", func(r chi.Router) {
		r.Route("/users/{discordid}/reviews", func(r1 chi.Router) {
			r1.Get("/", routes.GetReviews)
			r1.Put("/", routes.AddReview)
			r1.Delete("/", routes.DeleteReview)
		})
		r.HandleFunc("/users", routes.GetUserInfo)
		r.HandleFunc("/reports", routes.ReportReview)
		r.HandleFunc("/badges", routes.GetAllBadges)
		r.HandleFunc("/reviews", routes.SearchReview)
		r.HandleFunc("/blocks", routes.Blocks)
		r.HandleFunc("/settings", routes.Settings)
		r.HandleFunc("/notifications", routes.Notifications)
		r.HandleFunc("/settings", routes.Settings)
		r.Put("/appeals", routes.AppealReview)
	})

	mux.HandleFunc("/admins", routes.Admins)

	mux.Group(func(r chi.Router) {
		r.Use(httprate.LimitByRealIP(10, 1*time.Hour))

		r.HandleFunc("/api/reviewdb/authweb", routes.ReviewDBAuthWeb)
		r.HandleFunc("/api/reviewdb/auth", routes.ReviewDBAuth)
	})

	mux.Group(func(r chi.Router) {
		r.Use(routes.AdminMiddleware)

		r.Route(("/api/reviewdb/admin"), func(r chi.Router) {
			r.Get("/filters", routes.GetFilters)
			r.Put("/filters", routes.AddFilter)
			r.Delete("/filters", routes.DeleteFilter)
			r.Get("/reload", routes.ReloadConfig)
			r.Get("/reports", routes.GetReports)
			r.Get("/users", routes.GetUsersAdmin)
			r.Get("/users/{id}", routes.GetUserAdmin)
			r.Patch("/users", routes.PatchUserAdmin)
			r.Get("/badges", routes.GetAllBadges)
			r.Put("/badges", routes.AddBadge)
			r.Delete("/badges", routes.DeleteBadge)
		})
	})

	mux.Route("/api/reviewdb-twitter", func(r chi.Router) {
		r.HandleFunc("/auth", routes.ReviewDBTwitterAuth)
		r.HandleFunc("/users/{profileid}/reviews", routes.HandleTwitterRoutes)
		r.HandleFunc("/reports", routes.ReportTwitterReview)
	})

	mux.HandleFunc("/api/reviewdb/oauth/github", routes.LinkGithub)

	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "An Error occurred\n")
	})

	err = discord.SendLoggerWebhook(discord.WebhookData{
		Username: "ReviewDB Logger",
		Content:  "Starting Server...",
	})

	if err != nil {
		fmt.Println(err)
	}

	err = http.ListenAndServe(":"+common.Config.Port, mux)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
