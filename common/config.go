package common

import (
	"encoding/json"
	"fmt"
	"os"

	goaway "github.com/TwiN/go-away"
)

type ConfigStr struct {
	ApiEndpoint          string    `json:"api_endpoint"`
	DB                   *ConfigDB `json:"db"`
	RedirectUri          string    `json:"redirect_uri"`
	ClientId             string    `json:"client_id"`
	ClientSecret         string    `json:"client_secret"`
	GithubWebhookSecret  string    `json:"github_webhook_secret"`
	Origin               string    `json:"origin"`
	Port                 string    `json:"port"`
	BotToken             string    `json:"bot_token"`
	ReportWebhook        string    `json:"report_webhook"`
	AppealWebhook        string    `json:"appeal_webhook"`
	AdminToken           string    `json:"admin_token"`
	StupidityBotToken    string    `json:"stupidity_bot_token"`
	LoggerWebhook        string    `json:"logger_webhook"`
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
	LightProfanityDetector = goaway.NewProfanityDetector().WithCustomDictionary(Config.LightProfaneWordList, nil, nil)
}

func SaveConfig() {
	f, err := os.OpenFile("config.json", os.O_WRONLY, 0777)
	if err != nil {
		fmt.Println(err)
	}
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ")
	err = encoder.Encode(&Config)
	f.Close()
}

func init() {
	LoadConfig()
}
