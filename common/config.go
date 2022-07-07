package common

import (
	"encoding/json"
	"fmt"
	"os"
)

type ConfigStr struct {
	ApiEndpoint         string    `json:"api_endpoint"`
	DB                  *ConfigDB `json:"db"`
	RedirectUri         string    `json:"redirect_uri"`
	ClientId            string    `json:"client_id"`
	ClientSecret        string    `json:"client_secret"`
	GithubWebhookSecret string    `json:"github_webhook_secret"`
	Origin              string    `json:"origin"`
	Port                string    `json:"port"`
}

type ConfigDB struct {
	IP       string `json:"ip"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"db"`
}

var Config *ConfigStr

func GetConfig() *ConfigStr {
	if Config == nil {
		f, err := os.Open("config.json")
		if err != nil {
			fmt.Println(err)
		}

		err = json.NewDecoder(f).Decode(&Config)
		f.Close()
	}

	return Config
}
