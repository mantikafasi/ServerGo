package routes

import (
	"encoding/json"
	"io"
	"net/http"
	"server-go/modules"
	"strconv"
)

var StupidityDBAuth = func(w http.ResponseWriter, r *http.Request) {
	token, err := modules.AddStupidityDBUser(r.URL.Query().Get("code"))

	if err != nil {
		http.Redirect(w, r, "/error", http.StatusTemporaryRedirect)
		return
	}
	http.Redirect(w, r, "receiveToken/"+token, http.StatusTemporaryRedirect)
}

var Admins = func(w http.ResponseWriter, r *http.Request) {
	admins, err := modules.GetAdmins()
	if err != nil {
		io.WriteString(w, err.Error()+"\n")
		return
	}
	jsonAdmins, _ := json.Marshal(admins)
	io.WriteString(w, string(jsonAdmins))
}

var GetStupidity = func(w http.ResponseWriter, r *http.Request) {

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
}

var VoteStupidity = func(w http.ResponseWriter, r *http.Request) {

	var data modules.SDB_RequestData
	json.NewDecoder(r.Body).Decode(&data)

	res := modules.VoteStupidity(data.DiscordID, data.Token, data.Stupidity, data.SenderDiscordID)

	io.WriteString(w, res)
}
