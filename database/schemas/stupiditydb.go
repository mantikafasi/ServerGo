package schemas

import (
	"github.com/uptrace/bun"
)

type StupitStat struct {
	bun.BaseModel `bun:"table:stupidity_reviews"`

	ID                int32  `bun:"id,pk,autoincrement"`
	ReviewedDiscordID int64  `bun:"reviewed_discord_id,type:numeric"`
	StupidityValue    int32  `bun:"stupidity_value"`
	ReviewerDiscordID string `bun:"reviewer_discord_id,type:numeric"`
}

type UserInfo struct {
	bun.BaseModel `bun:"table:stupidity_users"`

	ID        int32  `bun:"id,pk,autoincrement"`
	DiscordID string `bun:"discord_id,type:numeric"`
	Token     string `bun:"token"`
}
