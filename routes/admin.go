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
	}{}

	response.ProfaneWords = common.Config.ProfaneWordList
	response.LightProfaneWords = common.Config.LightProfaneWordList

	json.NewEncoder(w).Encode(response)
}

const (
	ProfaneFilter      = "profane"
	LightProfaneFilter = "lightProfane"
)

func AddFilter(w http.ResponseWriter, r *http.Request) {

	data := struct {
		Word string `json:"word"`
		Type string `json:"type"`
	}{}

	json.NewDecoder(r.Body).Decode(&data)

	switch data.Type {
	case ProfaneFilter:
		common.Config.ProfaneWordList = append(common.Config.ProfaneWordList, data.Word)
	case LightProfaneFilter:
		common.Config.LightProfaneWordList = append(common.Config.LightProfaneWordList, data.Word)
	}
	common.SaveConfig()
}

func DeleteFilter(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Word string `json:"word"`
		Type string `json:"type"`
	}{}

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
	}
	common.SaveConfig()
}

func ReloadConfig(w http.ResponseWriter, r *http.Request) {
	common.LoadConfig()
}