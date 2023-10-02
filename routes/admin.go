package routes

import (
	"encoding/json"
	"net/http"
	"server-go/common"
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


func ReloadConfig(w http.ResponseWriter, r *http.Request) {
	common.LoadConfig()
}
