package routes

import (
	"encoding/json"
	"io"
	"net/http"
	"server-go/common"
	"server-go/modules"

	"github.com/go-chi/chi"
)

var HandleInteractions = func(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	signature := r.Header.Get("X-Signature-Ed25519")
	timestamp := r.Header.Get("X-Signature-Timestamp")

	message := append([]byte(timestamp), body...)
	if !common.VerifySignature(signature, message) {
		w.WriteHeader(401)
		return
	}
	var data modules.InteractionsData

	json.Unmarshal(message[len(timestamp):], &data)
	response, err := modules.Interactions(data)
	if err != nil {
		println(err.Error())
		w.WriteHeader(500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	io.WriteString(w, response)
}

var ReceiveToken = func(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	io.WriteString(w, "You have successfully logged in! Your token is: "+token+"\n\n You can now close this window.")
}
