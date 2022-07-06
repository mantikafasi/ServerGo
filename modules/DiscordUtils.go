package modules

import (
	"github.com/ravener/discord-oauth2"
	"golang.org/x/oauth2"
	"server-go/constantants"
)


func ExchangeCodePlus(token string,redirtect_uri string){
	conf := &oauth2.Config{
		Endpoint: discord.Endpoint,
		Scopes: []string{discord.ScopeIdentify},
		RedirectURL: redirtect_uri,
		ClientID: "id",
		ClientSecret: "secret",
	}
	
}

func ExchangeCode(token string){
	ExchangeCodePlus(token,constantants.REDIRECT_URI)
}