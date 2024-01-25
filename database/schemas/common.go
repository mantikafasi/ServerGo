package schemas

import (
	"time"

	"github.com/uptrace/bun"
)

type Oauth2Token struct {
	bun.BaseModel `bun:"table:oauth2_tokens"`

	Id           int32     `bun:"id,pk,autoincrement" json:"id"`
	UserId       int32     `bun:"user_id" json:"userId"`
	AccessToken  string    `bun:"access_token" json:"accessToken"`
	RefreshToken string    `bun:"refresh_token" json:"refreshToken"`
	Expiry       time.Time `bun:"expiry" json:"expiry"`
	Provider     string    `bun:"provider" json:"provider"`

	// this is probably not right place to put this but its better than creating another table
	Username   string `bun:"username" json:"username"`
	Avatar     string `bun:"avatar" json:"avatar"`
	ProviderId string `bun:"provider_id" json:"providerId"`
}
