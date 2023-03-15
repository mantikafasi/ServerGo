package common

import (
	"encoding/json"
	"fmt"
	"os"

	goaway "github.com/TwiN/go-away"
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
	BotToken            string    `json:"bot_token"`
	DiscordWebhook      string    `json:"discord_webhook"`
	AdminToken          string    `json:"admin_token"`
	StupidityBotToken   string    `json:"stupidity_bot_token"`
}

var ProfanityDetector *goaway.ProfanityDetector

type ConfigDB struct {
	IP       string `json:"ip"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"db"`
}

var Config *ConfigStr

var OptedOut []uint64

func init() {
	f, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err)
	}

	err = json.NewDecoder(f).Decode(&Config)
	f.Close()

	f2, er2 := os.Open("out.json") //this is list of users who opted out of reviewdb
	if er2 != nil {
		fmt.Println(er2)
	}

	er2 = json.NewDecoder(f2).Decode(&OptedOut)
	f2.Close()

	var profaneWords []string

	f3, er3 := os.Open("profanewordlist.json")
	if er3 != nil {
		fmt.Println(er3)
	}
	json.NewDecoder(f3).Decode(&profaneWords)
	ProfanityDetector = goaway.NewProfanityDetector().WithCustomDictionary(profaneWords, nil, nil)
}
