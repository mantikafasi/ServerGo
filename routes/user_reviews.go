package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"server-go/common"
	"server-go/database/schemas"
	"server-go/modules"
	"server-go/modules/discord"
	"server-go/modules/filtering"
	"strconv"
	"strings"
	"time"

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

type ReviewResponse struct {
	Response
	HasNextPage bool                 `json:"hasNextPage"`
	ReviewCount int                  `json:"reviewCount"`
	Reviews     []schemas.UserReview `json:"reviews"`
}

func AddUserReview(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Response
		Updated bool `json:"updated"`
	}{}

	var data modules.UR_RequestData
	json.NewDecoder(r.Body).Decode(&data)

	if chi.URLParam(r, "discordid") != "" {
		discordid, _ := strconv.ParseUint(chi.URLParam(r, "discordid"), 10, 64)
		data.DiscordID = discord.Snowflake(discordid)
	}

	if len(data.Comment) > 1000 {
		response.Message = "Comment Too Long"
	} else if len(strings.TrimSpace(data.Comment)) == 0 {
		response.Message = "Write Something Guh"
	}

	if slices.Contains(common.OptedOut, fmt.Sprint(data.DiscordID)) {
		response.Message = "This user opted out"
	}

	if response.Message != "" {
		common.SendStructResponse(w, response)
		return
	}

	if data.Token == common.Config.StartItBotToken {
		data.ReviewType = 4 // startit bot review type
	}

	reviewer, err := modules.GetDBUserViaTokenAndData(data.Token, data)

	if err != nil {
		Error(w, err)
		return
	}

	review := schemas.UserReview{
		ProfileID:    int64(data.DiscordID),
		ReviewerID:   reviewer.ID,
		Comment:      data.Comment,
		Type:         int32(data.ReviewType),
		TimestampStr: time.Now(),
	}

	for _, filterFunction := range filtering.ReviewDB {
		err = filterFunction(&reviewer, &review)

		if err != nil {
			Error(w, err)
			return
		}
	}

	res, err := modules.AddReview(&reviewer, &review)

	if err != nil {
		Error(w, err)
		println(err.Error())
	} else {
		response.Success = true
		response.Message = res
		if res == common.UPDATED {
			response.Updated = true
		}
	}

	common.SendStructResponse(w, response)
}

var ClientMods []string = []string{"aliucord", "betterdiscord", "powercordv2", "replugged", "enmity", "vencord", "vendetta"}

func ReviewDBAuth(w http.ResponseWriter, r *http.Request) {
	clientmod := r.URL.Query().Get("clientMod")
	if clientmod == "" {
		clientmod = "aliucord"
	}

	if !slices.Contains(ClientMods, clientmod) {
		w.WriteHeader(http.StatusPaymentRequired) // trolley
		common.SendStructResponse(w, ReviewDBAuthResponse{
			Success: false,
			Message: "Invalid clientMod",
		})
		return
	}

	token, err := modules.AddUserReviewsUser(r.URL.Query().Get("code"), clientmod, "/api/reviewdb/auth", r.Header.Get("CF-Connecting-IP"))

	if err != nil {
		println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
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

func ReviewDBAuthWeb(w http.ResponseWriter, r *http.Request) {
	token, err := modules.AddUserReviewsUser(r.URL.Query().Get("code"), "website", "/api/reviewdb/authweb", r.Header.Get("CF-Connecting-IP"))

	if err != nil {
		http.Redirect(w, r, common.WEBSITE+"/error ", http.StatusTemporaryRedirect)
		return
	}

	http.Redirect(w, r, common.WEBSITE+"/api/redirect?token="+url.QueryEscape(token), http.StatusTemporaryRedirect)
}

func ReportReview(w http.ResponseWriter, r *http.Request) {
	var data modules.UR_RequestData
	json.NewDecoder(r.Body).Decode(&data)

	response := Response{}

	if data.Token == "" || data.ReviewID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		response.Message = "Invalid Request"
		common.SendStructResponse(w, response)
		return
	}

	err := modules.ReportReview(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.Message = err.Error()
		common.SendStructResponse(w, response)
		return
	}
	response.Success = true
	response.Message = "Successfully Reported Review"
	common.SendStructResponse(w, response)
}

func DeleteReview(w http.ResponseWriter, r *http.Request) {
	var data modules.UR_RequestData //both reportdata and deletedata are same
	json.NewDecoder(r.Body).Decode(&data)

	responseData := Response{
		Success: false,
		Message: "",
	}

	if data.Token == "" || data.ReviewID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		responseData.Message = "Invalid Request"
		res, _ := json.Marshal(responseData)

		w.Write(res)
		return
	}

	err := modules.DeleteReviewWithData(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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

func GetReviews(w http.ResponseWriter, r *http.Request) {
	var userIDString string

	if r.URL.Query().Get("discordid") == "" {
		userIDString = chi.URLParam(r, "discordid")
	} else {
		userIDString = r.URL.Query().Get("discordid")
	}

	includeReviewsBy := r.URL.Query().Get("always_include_reviews_by")

	userID, _ := strconv.ParseInt(userIDString, 10, 64)
	flags64, _ := strconv.ParseInt(r.URL.Query().Get("flags"), 10, 32)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	flags := int32(flags64)

	var reviews []schemas.UserReview
	var err error

	response := ReviewResponse{}
	count := 0

	if slices.Contains(common.OptedOut, fmt.Sprint(userID)) {
		reviews = append([]schemas.UserReview{{
			ID: 0,
			Sender: schemas.Sender{
				ID:           0,
				Username:     "ReviewDB",
				ProfilePhoto: "https://cdn.discordapp.com/attachments/527211215785820190/1079358371481800725/c4b7353e759983f5a3d686c7937cfab7.png?size=128",
				DiscordID:    "287555395151593473",
				Badges:       []schemas.UserBadge{},
			}, Comment: "This user has opted out of ReviewDB. It means you cannot review this user.",
			Type: 3,
		}})
		response.Reviews = reviews
		response.Success = true
		common.SendStructResponse(w, response)
		return
	} else {
		options := modules.GetReviewsOptions{
			IncludeReviewsById: includeReviewsBy,
		}
		reviews, count, err = modules.GetReviewsWithOptions(userID, offset, options)

		response.ReviewCount = count
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.Success = false
		response.Message = err.Error()
		common.SendStructResponse(w, response)
		return
	}

	if len(reviews) == 51 {
		response.HasNextPage = true
		reviews = reviews[:len(reviews)-1]
	}

	for i, j := 0, len(reviews)-1; i < j; i, j = i+1, j-1 {
		reviews[i], reviews[j] = reviews[j], reviews[i]
	}

	/*
		if (len(reviews) > 8 && offset == 0) {
			var ix = random.Intn(len(reviews) - 1)
			reviews = append(reviews[:ix+1], reviews[ix:]...)
			reviews[ix] = schemas.UserReview{
				ID: 0,
				Sender: schemas.Sender{
					ID:           0,
					Username:     "ReviewDB",
					ProfilePhoto: "https://cdn.discordapp.com/attachments/527211215785820190/1079358371481800725/c4b7353e759983f5a3d686c7937cfab7.png?size=128",
					DiscordID:    "343383572805058560",
					Badges:       []schemas.UserBadge{},
				},
				Comment: "If you like ReviewDB try out ReviewDB Twitter at https://chrome.google.com/webstore/detail/reviewdb-twitter/kmgbgncbggoffjbefmnknffpofcajohj",
				Type: 3,
			}
		}
	*/

	if len(reviews) != 0 && !(flags&WarningFlag == WarningFlag) && offset == 0 {
		reviews = append([]schemas.UserReview{{
			ID:      0,
			Comment: "Spamming and writing offensive reviews will result with a ban. Please be respectful to other users.",
			Type:    3,
			Sender: schemas.Sender{
				DiscordID:    "1134864775000629298",
				ProfilePhoto: "https://cdn.discordapp.com/attachments/1045394533384462377/1084900598035513447/646808599204593683.png?size=128",
				Username:     "Warning",
				Badges: []schemas.UserBadge{
					{
						Name:        "Donor",
						Icon:        "https://cdn.discordapp.com/emojis/1084121193591885906.webp?size=96&quality=lossless",
						Description: "This badge is special to donors.",
						RedirectURL: "https://github.com/sponsors/mantikafasi",
						Type:        1,
					},
				},
			},
		}}, reviews...)
	}

	if reviews == nil { //we dont want to send null
		reviews = []schemas.UserReview{}
	}

	response.Reviews = reviews
	response.Success = true
	common.SendStructResponse(w, response)
}

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	//modules.GetLastReviewID(user.DiscordID)
	var data modules.UR_RequestData

	type UserInfo struct {
		schemas.URUser
		LastReviewID int32 `json:"lastReviewID"`
		UserType     int   `json:"type"`
	}

	token := r.Header.Get("Authorization")
	if token == "" {
		json.NewDecoder(r.Body).Decode(&data)
	} else {
		data = modules.UR_RequestData{
			Token: token,
		}
	}

	user, err := modules.GetDBUserViaToken(data.Token)
	response := UserInfo{user, modules.GetLastReviewID(user.DiscordID), int(user.Type)}
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	dbBadges := modules.GetBadgesOfUser(user.DiscordID)
	badges := make([]schemas.UserBadge, len(dbBadges))
	for i, b := range dbBadges {
		badges[i] = schemas.UserBadge(b)
	}

	response.Badges = badges

	json.NewEncoder(w).Encode(response)
}

func GetAllBadges(w http.ResponseWriter, r *http.Request) {
	type UserBadge struct {
		schemas.UserBadge
		DiscordID string `json:"discordID"`
	}

	legacyBadges, err := modules.GetAllBadges()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	badges := make([]UserBadge, len(legacyBadges))
	for i, b := range legacyBadges {
		badges[i] = UserBadge{schemas.UserBadge(b), b.TargetDiscordID}
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

func SearchReview(w http.ResponseWriter, r *http.Request) {
	type SearchRequestData struct {
		Query string `json:"query"`
		Token string `json:"token"`
	}
	response := ReviewResponse{}

	var data SearchRequestData
	json.NewDecoder(r.Body).Decode(&data)

	reviews, err := modules.SearchReviews(data.Query, data.Token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.Success = false
		response.Message = err.Error()
		common.SendStructResponse(w, response)
		return
	}

	response.Success = true
	response.Reviews = reviews
	response.Message = "Success"

	common.SendStructResponse(w, response)
}

func Settings(w http.ResponseWriter, r *http.Request) {

	token := r.Header.Get("Authorization")

	if token == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var settings modules.Settings

	json.NewDecoder(r.Body).Decode(&settings)

	user, err := modules.GetDBUserViaToken(token)

	settings.DiscordID = user.DiscordID

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case "GET":
		settings, err := modules.GetSettings(user.DiscordID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("err: %v\n", err)
			return
		}
		json.NewEncoder(w).Encode(settings)

	case "PATCH":
		err := modules.SetSettings(settings)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("err: %v\n", err)
			return
		}
		w.WriteHeader(200)
	}

	optedOutUsers, err := modules.GetOptedOutUsers()
	if err != nil {
		fmt.Println(err)
	}
	common.OptedOut = optedOutUsers
}

func AppealReview(w http.ResponseWriter, r *http.Request) {
	appealRequest := schemas.ReviewDBAppeal{}

	user, err := Authorize(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !user.IsBanned() {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewDecoder(r.Body).Decode(&appealRequest)

	appealRequest.UserID = user.ID
	appealRequest.BanID = user.BanID

	err = modules.AppealBan(appealRequest, user)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusOK)
}

/*
func AppealReview(w http.ResponseWriter,r *http.Request) {
	var user,err := common.Authorize(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

}
*/
