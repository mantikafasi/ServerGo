package routes

import (
	"net/http"
	"server-go/common"
	"server-go/modules"
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
