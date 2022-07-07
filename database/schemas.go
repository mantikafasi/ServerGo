package database

import (
	"context"

	"github.com/uptrace/bun"
)

type StupitStat struct {
	bun.BaseModel `bun:"table:stupit_table"`

	ID        int32  `bun:"id,pk,autoincrement"`
	DiscordID string `bun:"discordid,type:numeric"`
	Stupidity int32  `bun:"stupidity"`
	SenderID  string `bun:"senderdiscordid,type:numeric"`
}

type UserInfo struct {
	bun.BaseModel `bun:"table:user_info"`

	ID        int32  `bun:"id,pk,autoincrement"`
	DiscordID string `bun:"discordid,type:numeric"`
	Token     string `bun:"token"`
}

type UserReview struct {
	bun.BaseModel `bun:"table:userreviews"`

	ID           int32  `bun:"id,pk,autoincrement" json:"id"`
	UserID       string `bun:"userid,type:numeric" json:"userid"`
	Star         int32  `bun:"star" json:"star"`
	SenderUserID int32  `bun:"senderuserid" json:"senderuserid"`
	Comment      string `bun:"comment" json:"comment"`

	User            *URUser `bun:"rel:belongs-to,join:senderuserid=id" json:"-"`
	SenderDiscordID string  `bun:"-" json:"senderdiscordid"`
	SenderUsername  string  `bun:"-" json:"username"`
}

type URUser struct {
	bun.BaseModel `bun:"table:ur_users"`

	ID        int32  `bun:"id,pk,autoincrement" json:"id"`
	DiscordID string `bun:"discordid,type:numeric" json:"discordid"`
	Token     string `bun:"token" json:"token"`
	Username  string `bun:"username" json:"username"`
}

func createSchema() error {
	models := []any{
		(*StupitStat)(nil),
		(*UserInfo)(nil),
		(*UserReview)(nil),
		(*URUser)(nil),
	}

	for _, model := range models {
		if _, err := DB.NewCreateTable().IfNotExists().Model(model).Exec(context.Background()); err != nil {
			return err
		}
	}
	return nil
}
