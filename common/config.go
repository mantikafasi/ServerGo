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
	DB                    *ConfigDB `json:"db"`
	GithubWebhookSecret   string    `json:"github_webhook_secret"`
	Origin                string    `json:"origin"`
	Port                  string    `json:"port"`
	BotToken              string    `json:"bot_token"`
	ReportWebhook         string    `json:"report_webhook"`
	JunkReportWebhook     string    `json:"junk_report_webhook"`
	AppealWebhook         string    `json:"appeal_webhook"`
	AdminToken            string    `json:"admin_token"`
	StartItBotToken       string    `json:"start_it_bot_token"`
	LoggerWebhook         string    `json:"logger_webhook"`
	Discord               *Client   `json:"discord"`
	Twitter               *Client   `json:"twitter"`
	Github                *Client   `json:"github"`
	Debug                 bool      `json:"debug"`
	CommentAnalyzerAPIKey string    `json:"comment_analyzer_api_key"`
	ProfaneWordList       []string  `json:"profane_word_list"`
	LightProfaneWordList  []string  `json:"light_profane_word_list"`
	BanWordList           []string  `json:"ban_word_list"`
}

var LightProfanityDetector *goaway.ProfanityDetector
var ProfanityDetector *goaway.ProfanityDetector
var BanWordDetector *goaway.ProfanityDetector

type ConfigDB struct {
	IP        string `json:"ip"`
	User      string `json:"user"`
	Password  string `json:"password"`
	Name      string `json:"db"`
	UseSocket bool   `json:"use_socket"`
}

type GoodPersonConfigStr struct {
	BadNouns         []string `json:"bad_nouns"`
	BadVerbs         []string `json:"bad_verbs"`
	ReplacementNouns []string `json:"replacement_nouns"`
	ReplacementVerbs []string `json:"replacement_verbs"`
}

var Config *ConfigStr

var GoodPersonConfig *GoodPersonConfigStr

var OptedOut []string

func LoadConfig() {
	f, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err)
	}

	_ = json.NewDecoder(f).Decode(&Config)
	f.Close()

	f, err = os.Open("good_person.json")
	if err != nil {
		fmt.Println(err)
	}

	_ = json.NewDecoder(f).Decode(&GoodPersonConfig)

	ProfanityDetector = goaway.NewProfanityDetector().WithCustomDictionary(Config.ProfaneWordList, nil, nil)
	LightProfanityDetector = goaway.NewProfanityDetector().WithCustomDictionary(Config.LightProfaneWordList, nil, nil)
	BanWordDetector = goaway.NewProfanityDetector().WithCustomDictionary(Config.BanWordList, nil, nil)
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
