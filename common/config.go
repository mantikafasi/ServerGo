package common

import(
	"encoding/json"
	"fmt"
	"os"
)

type ConfigStr struct {
	API_ENDPOINT string `json:"api_endpoint"`
	DBIP string `json:"dbip"`
	REDIRECT_URI string `json:"redirect_uri"`
	CLIENT_ID string `json:"client_id"`
	CLIENT_SECRET string `json:"client_secret"`
	DBUSER string `json:"db_user"`
	DBPASSWORD string `json:"db_password"`
	DBNAME string `json:"db_name"`
	GITHUB_WEBHOOK_SECRET string `json:"github_webhook_secret"`
}

var Config *ConfigStr

func GetConfig() *ConfigStr {
	if Config==nil{
		f, err := os.Open("config.json")
		if err != nil {
			fmt.Println(err)
		}
		
		err = json.NewDecoder(f).Decode(&Config)
		f.Close()
	}

	return Config
}