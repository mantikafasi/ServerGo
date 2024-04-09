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
		if err == sql.ErrNoRows{ 
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	common.SendStructResponse(w, user)
}

func AddBadge(w http.ResponseWriter, r *http.Request) {
	var badge schemas.UserBadge
	json.NewDecoder(r.Body).Decode(&badge)

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