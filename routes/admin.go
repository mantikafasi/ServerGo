package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"server-go/common"
	"server-go/database/schemas"
	"server-go/modules"

	"github.com/go-chi/chi/v5"
)

func GetFilters(w http.ResponseWriter, r *http.Request) {
	response := struct {
		ProfaneWords      []string `json:"profaneWords"`
		LightProfaneWords []string `json:"lightProfaneWords"`
		BanWords          []string `json:"banWords"`
	}{}

	response.ProfaneWords = common.Config.ProfaneWordList
	response.LightProfaneWords = common.Config.LightProfaneWordList
	response.BanWords = common.Config.BanWordList

	json.NewEncoder(w).Encode(response)
}

const (
	ProfaneFilter      = "profane"
	LightProfaneFilter = "lightProfane"
	BanFilter          = "ban"
)

type FilterStruct struct {
	Word string `json:"word"`
	Type string `json:"type"`
}

func AddFilter(w http.ResponseWriter, r *http.Request) {

	var data FilterStruct

	json.NewDecoder(r.Body).Decode(&data)

	switch data.Type {
	case ProfaneFilter:
		common.Config.ProfaneWordList = append(common.Config.ProfaneWordList, data.Word)
	case LightProfaneFilter:
		common.Config.LightProfaneWordList = append(common.Config.LightProfaneWordList, data.Word)
	case BanFilter:
		common.Config.BanWordList = append(common.Config.BanWordList, data.Word)
	}

	common.SaveConfig()
	common.LoadConfig()
	w.WriteHeader(http.StatusOK)
}

func DeleteFilter(w http.ResponseWriter, r *http.Request) {
	var data FilterStruct

	json.NewDecoder(r.Body).Decode(&data)
	switch data.Type {
	case ProfaneFilter:
		for i, word := range common.Config.ProfaneWordList {
			if word == data.Word {
				common.Config.ProfaneWordList = append(common.Config.ProfaneWordList[:i], common.Config.ProfaneWordList[i+1:]...)
				break
			}
		}
	case LightProfaneFilter:
		for i, word := range common.Config.LightProfaneWordList {
			if word == data.Word {
				common.Config.LightProfaneWordList = append(common.Config.LightProfaneWordList[:i], common.Config.LightProfaneWordList[i+1:]...)
				break
			}
		}
	case BanFilter:
		for i, word := range common.Config.BanWordList {
			if word == data.Word {
				common.Config.BanWordList = append(common.Config.BanWordList[:i], common.Config.BanWordList[i+1:]...)
				break
			}
		}
	}
	common.SaveConfig()
	common.LoadConfig()
}

func GetReports(w http.ResponseWriter, r *http.Request) {
	limit := common.GetIntQueryOrDefault(r, "limit", 50)
	offset := common.GetIntQueryOrDefault(r, "offset", 0)

	reports, err := modules.GetReports(offset, limit)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	common.SendStructResponse(w, reports)
}

func ReloadConfig(w http.ResponseWriter, r *http.Request) {
	common.LoadConfig()
}

func GetUsersAdmin(w http.ResponseWriter, r *http.Request) {
	limit := common.GetIntQueryOrDefault(r, "limit", 50)
	offset := common.GetIntQueryOrDefault(r, "offset", 0)
	query := r.URL.Query().Get("query")
	ip_hash := common.GetQueryOrDefault(r, "ip_hash", "")

	err, users := modules.GetUsersAdmin(query, limit, offset, ip_hash)

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	common.SendStructResponse(w, users)
}

func PatchUserAdmin(w http.ResponseWriter, r *http.Request) {
	var user schemas.ReviewDBUserFull
	json.NewDecoder(r.Body).Decode(&user)

	err := modules.PatchUserAdmin(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func GetUserAdmin(w http.ResponseWriter, r *http.Request) {
	// this id can be either discorid and reviewdb id
	id := chi.URLParam(r, "id")

	user, err := modules.GetUserAdmin(id)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	common.SendStructResponse(w, user)
}

func GetUserReviewsAdmin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := modules.GetUserAdmin(id)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	limit := common.GetIntQueryOrDefault(r, "limit", 50)
	offset := common.GetIntQueryOrDefault(r, "offset", 0)
	if limit < 1 || limit > 100 || offset < 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	reviews, count, err := modules.GetReviewsByReviewerAdmin(user.ID, offset, limit)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	common.SendStructResponse(w, struct {
		Reviews     []schemas.UserReview `json:"reviews"`
		ReviewCount int                  `json:"reviewCount"`
	}{Reviews: reviews, ReviewCount: count})
}

func BanUserAdmin(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Days     int32 `json:"days"`
		ReviewID int32 `json:"reviewID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil || (data.Days != 1 && data.Days != 3 && data.Days != 7 && data.Days != 30) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := modules.GetUserAdmin(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	review := schemas.UserReview{}
	if data.ReviewID != 0 {
		review, err = modules.GetReview(data.ReviewID)
		if err != nil || review.Sender.DiscordID != user.DiscordID {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	if err := modules.BanUser(user.DiscordID, r.Header.Get("Authorization"), data.Days, review); err != nil {
		common.SendStructResponse(w, Response{Success: false, Message: err.Error()})
		return
	}
	common.SendStructResponse(w, Response{Success: true, Message: "Review author banned"})
}

func AddBadge(w http.ResponseWriter, r *http.Request) {
	var data struct {
		TargetDiscordID string `json:"targetDiscordID"`
		Name            string `json:"name"`
		Icon            string `json:"icon"`
		RedirectURL     string `json:"redirectURL"`
		Type            int32  `json:"type"`
		Description     string `json:"description"`
	}
	json.NewDecoder(r.Body).Decode(&data)
	badge := schemas.UserBadge{
		TargetDiscordID: data.TargetDiscordID,
		Name:            data.Name,
		Icon:            data.Icon,
		RedirectURL:     data.RedirectURL,
		Type:            data.Type,
		Description:     data.Description,
	}

	err := modules.AddBadge(badge)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteBadge(w http.ResponseWriter, r *http.Request) {
	id := common.GetQueryOrDefault(r, "id", "")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := modules.DeleteBadge(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
