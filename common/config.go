package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	goaway "github.com/TwiN/go-away"
)

const (
	WEBSITE = "https://reviewdb.mantikafasi.dev"
)

type Client struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	ApiEndpoint  string `json:"api_endpoint"`
}

type ConfigStr struct {
	DB                   *ConfigDB `json:"db"`
	GithubWebhookSecret  string    `json:"github_webhook_secret"`
	Origin               string    `json:"origin"`
	Port                 string    `json:"port"`
	BotToken             string    `json:"bot_token"`
	ReportWebhook        string    `json:"report_webhook"`
	AppealWebhook        string    `json:"appeal_webhook"`
	AdminToken           string    `json:"admin_token"`
	StartItBotToken      string    `json:"start_it_bot_token"`
	LoggerWebhook        string    `json:"logger_webhook"`
	Discord              *Client   `json:"discord"`
	Twitter              *Client   `json:"twitter"`
	Debug                bool      `json:"debug"`
	ProfaneWordList      []string  `json:"profane_word_list"`
	LightProfaneWordList []string  `json:"light_profane_word_list"`
}

var LightProfanityDetector *goaway.ProfanityDetector
var ProfanityDetector *goaway.ProfanityDetector

type ConfigDB struct {
	IP       string `json:"ip"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"db"`
}

var Config *ConfigStr

var OptedOut []string

func LoadConfig() {
	f, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err)
	}

	err = json.NewDecoder(f).Decode(&Config)
	f.Close()

	ProfanityDetector = goaway.NewProfanityDetector().WithCustomDictionary(Config.ProfaneWordList, nil, nil)
	LightProfanityDetector = goaway.NewProfanityDetector().WithCustomDictionary(Config.LightProfaneWordList, nil,nil)
}

func SaveConfig() {

	data, err := json.MarshalIndent(*Config, "", "  ")

	err = ioutil.WriteFile("config.json", data, 0644)
	if err != nil {
		fmt.Println(err)
	}

}

func init() {
	LoadConfig()
}
