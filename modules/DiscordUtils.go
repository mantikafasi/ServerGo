package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"server-go/constantants"

	"github.com/ravener/discord-oauth2"
	"golang.org/x/oauth2"
)

type DiscordUser struct {
	ID int64 `json:"id"`
	Username string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar string `json:"avatar"`
}


func ExchangeCodePlus(code string,redirtect_uri string) (string,error){
	conf := &oauth2.Config{
		Endpoint: discord.Endpoint,
		Scopes: []string{discord.ScopeIdentify},
		RedirectURL: redirtect_uri,
		ClientID: constantants.CLIENT_ID,
		ClientSecret: constantants.CLIENT_SECRET,
	}

	token, err := conf.Exchange(context.Background(),code)
	if err != nil {
		return "", err
	} else {
		return token.AccessToken, nil
	}

}

func GetUser(token string) (DiscordUser,error) {
	body, _ := json.Marshal(map[string]string{
		"Authorization": "Bearer " + token, })
	req,_ := http.NewRequest("GET", "https://discordapp.com/api/users/@me", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return DiscordUser{}, err
	}

	defer resp.Body.Close()
	body, _ = ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
	var user DiscordUser
	json.Unmarshal(body, &user)

	return user,nil

}

func ExchangeCode(token string) (string,error) {
	
	return ExchangeCodePlus(token,constantants.REDIRECT_URI)
}