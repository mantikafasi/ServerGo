package legacy_routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"server-go/common"
	"server-go/database"
	"server-go/modules"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

type UR_AuthResponse struct {
	Token  string `json:"token"`
	Status int32  `json:"status"`
}

type Response struct {
	Successful bool   `json:"successful"`
	Message    string `json:"message"`
}

var AddUserReview = func(w http.ResponseWriter, r *http.Request) {
	var data modules.UR_RequestData
	json.NewDecoder(r.Body).Decode(&data)

	if len(data.Comment) > 1000 {
		io.WriteString(w, "Comment Too Long")
		return

	} else if len(strings.TrimSpace(data.Comment)) == 0 {
		io.WriteString(w, "Write Something Guh")
		return
	}

	if slices.Contains(common.OptedOut, string(data.DiscordID)) {
		io.WriteString(w, "This user opted out")
		return
	}

	res, err := modules.AddReview(data)
	if err != nil {
		println(err.Error())
	}
	io.WriteString(w, res)
}

var ReviewDBAuth = func(w http.ResponseWriter, r *http.Request) {
	clientmod := r.URL.Query().Get("clientMod")
	if clientmod == "" {
		clientmod = "aliucord"
	}

	token, err := modules.AddUserReviewsUser(r.URL.Query().Get("code"), clientmod, "")

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

var GetReviews = func(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(r.URL.Query().Get("discordid"), 10, 64)

	if slices.Contains(common.OptedOut, fmt.Sprint(userID)) {
		reviews := append([]database.UserReview{{
			ID:              0,
			SenderUsername:  "ReviewDB",
			ProfilePhoto:    "https://cdn.discordapp.com/attachments/527211215785820190/1079358371481800725/c4b7353e759983f5a3d686c7937cfab7.png?size=128",
			Comment:         "This user has opted out of ReviewDB. It means you cannot review this user.",
			ReviewType:      1,
			SenderDiscordID: "287555395151593473",
			SystemMessage:   true,
			Badges:          []database.UserBadgeLegacy{},
		}})
		jsonReviews, _ := json.Marshal(reviews)

		io.WriteString(w, string(jsonReviews))
		return
	}

	reviews, err := modules.GetReviewsLegacy(userID)

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
			ProfilePhoto:    "https://cdn.discordapp.com/attachments/527211215785820190/1079358371481800725/c4b7353e759983f5a3d686c7937cfab7.png?size=128",
			Comment:         "If you like the plugins I make, please consider supporting me at: \nhttps://github.com/sponsors/mantikafasi\n You can disable this in settings",
			ReviewType:      1,
			SenderDiscordID: "287555395151593473",
			SystemMessage:   true,
		}}, reviews...)
	}

	if len(reviews) != 0 {
		reviews = append([]database.UserReview{{
			ID:              0,
			SenderUsername:  "Warning",
			ProfilePhoto:    "https://cdn.discordapp.com/attachments/1045394533384462377/1084900598035513447/646808599204593683.png?size=128",
			Comment:         "Spamming and writing offensive reviews will result with a ban. Please be respectful to other users.",
			ReviewType:      1,
			SenderDiscordID: "287555395151593473",
			SystemMessage:   true,
			Badges:          []database.UserBadgeLegacy{},
		}}, reviews...)
	}

	jsonReviews, _ := json.Marshal(reviews)
	reviewsStr := string(jsonReviews)

	if reviewsStr == "null" {
		reviewsStr = "[]"
	}

	io.WriteString(w, reviewsStr)
}
