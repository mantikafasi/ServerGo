package routes

import (
	"encoding/json"
	"io"
	"net/http"
	"server-go/common"
	"server-go/database"
	"server-go/modules"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"golang.org/x/exp/slices"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type ReviewDBAuthResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token"`
}

var AddUserReview = func(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Response
		Updated bool `json:"updated"`
	}{}

	var data modules.UR_RequestData
	json.NewDecoder(r.Body).Decode(&data)

	if chi.URLParam(r, "discordid") != "" {
		discordid, _ := strconv.ParseUint(chi.URLParam(r, "discordid"), 10, 64)
		data.DiscordID = modules.Snowflake(discordid)
	}

	if len(data.Comment) > 1000 {
		response.Message = "Comment Too Long"
	} else if len(strings.TrimSpace(data.Comment)) == 0 {
		response.Message = "Write Something Guh"
	}

	if slices.Contains(common.OptedOut, uint64(data.DiscordID)) {
		response.Message = "This user opted out"
	}

	if response.Message != "" {
		common.SendStructResponse(w, response)
		return
	}

	res, err := modules.AddReview(data.DiscordID, data.Token, data.Comment, int32(data.ReviewType))
	if err != nil {
		response.Success = false
		response.Message = err.Error()
		println(err.Error())
	} else {
		response.Success = true
		response.Message = res
		if res == "Updated your review" { // I will fix this once I delete old api
			response.Updated = true
		}
	}

	common.SendStructResponse(w, response)
}

var ClientMods []string = []string{"aliucord", "betterdiscord", "powercordv2", "replugged", "enmity", "vencord", "vendetta"}

var ReviewDBAuth = func(w http.ResponseWriter, r *http.Request) {
	clientmod := r.URL.Query().Get("clientMod")
	if clientmod == "" {
		clientmod = "aliucord"
	}

	if !slices.Contains(ClientMods, clientmod) {
		common.SendStructResponse(w, ReviewDBAuthResponse{
			Success: false,
			Message: "Invalid clientMod",
		})
		return
	}

	token, err := modules.AddUserReviewsUser(r.URL.Query().Get("code"), clientmod, "/api/reviewdb/auth")

	if err != nil {
		io.WriteString(w, `{"token": "", "success": false}`)
		return
	}

	res := ReviewDBAuthResponse{
		Token:   token,
		Success: true,
	}

	response, _ := json.Marshal(res)
	io.WriteString(w, string(response))
}

var ReportReview = func(w http.ResponseWriter, r *http.Request) {
	var data modules.ReportData
	json.NewDecoder(r.Body).Decode(&data)

	response := Response{}

	if data.Token == "" || data.ReviewID == 0 {
		response.Message = "Invalid Request"
		common.SendStructResponse(w, response)
		return
	}

	err := modules.ReportReview(data.ReviewID, data.Token)
	if err != nil {
		response.Message = err.Error()
		common.SendStructResponse(w, response)
		return
	}
	response.Success = true
	response.Message = "Successfully Reported Review"
	common.SendStructResponse(w, response)
}

var DeleteReview = func(w http.ResponseWriter, r *http.Request) {
	var data modules.ReportData //both reportdata and deletedata are same
	json.NewDecoder(r.Body).Decode(&data)

	responseData := Response{
		Success: false,
		Message: "",
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
	responseData.Success = true
	responseData.Message = "Successfully Deleted Review"
	res, _ := json.Marshal(responseData)
	w.Write(res)
}

const (
	AdFlag      = 0b00000001
	WarningFlag = 0b00000010
)

var GetReviews = func(w http.ResponseWriter, r *http.Request) {
	type ReviewResponse struct {
		Response
		Reviews []modules.UserReview `json:"reviews"`
	}

	var userIDString string

	if r.URL.Query().Get("discordid") == "" {
		userIDString = chi.URLParam(r, "discordid")
	} else {
		userIDString = r.URL.Query().Get("discordid")
	}

	userID, _ := strconv.ParseInt(userIDString, 10, 64)
	flags64, _ := strconv.ParseInt(r.URL.Query().Get("flags"), 10, 32)
	flags := int32(flags64)

	reviews, err := modules.GetReviews(userID)
	response := ReviewResponse{}

	if slices.Contains(common.OptedOut, uint64(userID)) {
		reviews := append([]database.UserReview{{
			ID:              0,
			SenderUsername:  "ReviewDB",
			ProfilePhoto:    "https://cdn.discordapp.com/attachments/527211215785820190/1079358371481800725/c4b7353e759983f5a3d686c7937cfab7.png?size=128",
			Comment:         "This user has opted out of ReviewDB. It means you cannot review this user.",
			ReviewType:      3,
			SenderDiscordID: "287555395151593473",
			Badges:          []database.UserBadgeLegacy{},
		}})
		jsonReviews, _ := json.Marshal(reviews)

		io.WriteString(w, string(jsonReviews))
		return
	}

	for i, j := 0, len(reviews)-1; i < j; i, j = i+1, j-1 {
		reviews[i], reviews[j] = reviews[j], reviews[i]
	}

	if err != nil {
		response.Success = false
		response.Message = err.Error()
		common.SendStructResponse(w, response)
		return
	}

	if r.Header.Get("User-Agent") == "Aliucord (https://github.com/Aliucord/Aliucord)" && flags&AdFlag == AdFlag {
		reviews = append([]modules.UserReview{{
			Comment:    "If you like the plugins I make, please consider supporting me at: \nhttps://github.com/sponsors/mantikafasi\n You can disable this in settings",
			ReviewType: 2,
			Sender: modules.Sender{
				DiscordID:    "287555395151593473",
				ProfilePhoto: "https://cdn.discordapp.com/attachments/527211215785820190/1079358371481800725/c4b7353e759983f5a3d686c7937cfab7.png?size=128",
				Username:     "ReviewDB",
			},
		}}, reviews...)
	}

	if len(reviews) != 0 && !(flags&WarningFlag == WarningFlag) {
		reviews = append([]modules.UserReview{{
			ID:         0,
			Comment:    "Spamming and writing offensive reviews will result with a ban. Please be respectful to other users.",
			ReviewType: 3,
			Sender: modules.Sender{
				DiscordID:    "287555395151593473",
				ProfilePhoto: "https://cdn.discordapp.com/attachments/1045394533384462377/1084900598035513447/646808599204593683.png?size=128",
				Username:     "Warning",
				Badges:       []database.UserBadge{},
			},
		}}, reviews...)
	}

	if reviews == nil { //we dont want to send null
		reviews = []modules.UserReview{}
	}

	response.Reviews = reviews
	response.Success = true
	common.SendStructResponse(w, response)
}

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	var data modules.UR_RequestData
	json.NewDecoder(r.Body).Decode(&data)

	user, err := modules.GetDBUserViaToken(data.Token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	dbBadges := modules.GetBadgesOfUser(user.DiscordID)
	badges := make([]database.UserBadge, len(dbBadges))
	for i, b := range dbBadges {
		badges[i] = database.UserBadge(b)
	}

	user.Badges = badges

	json.NewEncoder(w).Encode(user)
}

func GetAllBadges(w http.ResponseWriter, r *http.Request) {
	type UserBadge struct {
		database.UserBadge
		DiscordID string `json:"discordID"`
	}

	legacyBadges, err := modules.GetAllBadges()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	badges := make([]UserBadge, len(legacyBadges))
	for i, b := range legacyBadges {
		badges[i] = UserBadge{database.UserBadge(b), b.DiscordID}
	}
	json.NewEncoder(w).Encode(badges)
}

var HandleReviews = func(w http.ResponseWriter, r *http.Request) {
	method := r.Method

	switch method {
	case "GET":
		GetReviews(w, r)
	case "PUT":
		AddUserReview(w, r)
	case "DELETE":
		DeleteReview(w, r)
	case "REPORT":
		ReportReview(w, r)
	}
}
