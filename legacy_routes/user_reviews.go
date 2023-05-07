package legacy_routes

import (
	"encoding/json"
	"io"
	"net/http"
	"server-go/modules"
)

type UR_AuthResponse struct {
	Token  string `json:"token"`
	Status int32  `json:"status"`
}

type Response struct {
	Successful bool   `json:"successful"`
	Message    string `json:"message"`
}

var ReviewDBAuth = func(w http.ResponseWriter, r *http.Request) {
	clientmod := r.URL.Query().Get("clientMod")
	if clientmod == "" {
		clientmod = "aliucord"
	}

	token, err := modules.AddUserReviewsUser(r.URL.Query().Get("code"), clientmod, "",r.Header.Get("CF-Connecting-IP")  )

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
}