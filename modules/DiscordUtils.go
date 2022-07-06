package modules

import (
	"bytes"
	"context"
	"encodings/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"server-go/constantants"

	"github.com/ravener/discord-oauth2"
	"golang.org/x/oauth2"
)


func ExchangeCodePlus(code string,redirtect_uri string) (oauth2.Token,error){
	conf := &oauth2.Config{
		Endpoint: discord.Endpoint,
		Scopes: []string{discord.ScopeIdentify},
		RedirectURL: redirtect_uri,
		ClientID: constantants.CLIENT_ID,
		ClientSecret: constantants.CLIENT_SECRET,
	}

	token, err := conf.Exchange(context.Background(),code)
	if err != nil {
		return oauth2.Token{}, err
	} else {
		return *token, nil
	}

}

func GetUser(token string) {
	body, _ := json.Marshal(map[string]string{
		"Authorization": "Bearer " + token, })
	resp, _ := http.Post(constantants.API_ENDPOINT+"/users/@me","application/json", bytes.NewBuffer(body))
	
	defer resp.Body.Close()

	var jason map[string]interface{}

	body,_ = ioutil.ReadAll(resp.Body)
	json.unmarshal(body,&jason)

	fmt.Println(jason["id"])
}

func ExchangeCode(token string){
	ExchangeCodePlus(token,constantants.REDIRECT_URI)
}