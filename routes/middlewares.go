package routes

import (
	"context"
	"net/http"
	"server-go/common"
	"server-go/database"
	"server-go/database/schemas"
	"server-go/modules"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func AdminMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var token = r.Header.Get("Authorization")

		if token == common.Config.AdminToken {
			handler.ServeHTTP(w, r)
			return
		}
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		user, err := modules.GetDBUserViaToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if user.IsAdmin() {
			handler.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func CorsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// ReviewExistsMiddleware checks that the {reviewid} URL param refers to an existing review.
// Returns 404 immediately if it does not exist or is not a valid integer.
func ReviewMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "reviewid")
		id, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil || id <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			common.SendStructResponse(w, Response{Message: "Invalid review ID"})
			return
		}

		count, err := database.DB.NewSelect().
			Model((*schemas.UserReview)(nil)).
			Where("id = ?", int32(id)).
			Count(context.Background())
		if err != nil || count == 0 {
			w.WriteHeader(http.StatusNotFound)
			common.SendStructResponse(w, Response{Message: "Review not found"})
			return
		}

		handler.ServeHTTP(w, r)
	})
}
