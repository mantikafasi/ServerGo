package database

import (
	"context"

	"github.com/uptrace/bun"
)

type StupitStat struct {
	bun.BaseModel `bun:"table:stupit_table"`

	ID        int32  `bun:"id,pk,autoincrement"`
	DiscordID int64  `bun:"discordid,type:numeric"`
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
	UserID       int64  `bun:"userid,type:numeric" json:"-"`
	Star         int32  `bun:"star" json:"star"`
	SenderUserID int32  `bun:"senderuserid" json:"senderuserid"`
	Comment      string `bun:"comment" json:"comment"`
	ReviewType   int32  `bun:"reviewtype" json:"reviewtype"`

	User            *URUser     `bun:"rel:belongs-to,join:senderuserid=id" json:"-"`
	SenderDiscordID string      `bun:"-" json:"senderdiscordid"`
	SenderUsername  string      `bun:"-" json:"username"`
	ProfilePhoto    string      `bun:"-" json:"profile_photo"`
	Badges          []UserBadge `bun:"-" json:"badges"`
}

type URUser struct {
	bun.BaseModel `bun:"table:ur_users"`

	ID           int32  `bun:"id,pk,autoincrement" json:"id"`
	DiscordID    string `bun:"discordid,type:numeric" json:"discordid"`
	Token        string `bun:"token" json:"token"`
	Username     string `bun:"username" json:"username"`
	UserType     int32  `bun:"column:type" json:"-"`
	ProfilePhoto string `bun:"profile_photo" json:"profile_photo"`
	ClientMod    string `bun:"client_mod" json:"client_mod"`
}

type ReviewReport struct {
	bun.BaseModel `bun:"table:ur_reports"`

	ID         int32 `bun:"id,pk,autoincrement"`
	UserID     int32 `bun:"userid"`
	ReviewID   int32 `bun:"reviewid"`
	ReporterID int32 `bun:"reporterid"`
}

type UserBadge struct {
	bun.BaseModel `bun:"table:userbadges"`

	ID               int32  `bun:"id,pk,autoincrement" json:"-"`
	DiscordID        string `bun:"discordid,type:numeric" json:"-"`
	BadgeName        string `bun:"badge_name" json:"badge_name"`
	BadgeIcon        string `bun:"badge_icon" json:"badge_icon"`
	RedirectURL      string `bun:"redirect_url" json:"redirect_url"`
	BadgeType        int32  `bun:"badge_type" json:"badge_type"`
	BadgeDescription string `bun:"badge_description" json:"badge_description"`
}

func createSchema() error {
	models := []any{
		(*StupitStat)(nil),
		(*UserInfo)(nil),
		(*UserReview)(nil),
		(*URUser)(nil),
		(*ReviewReport)(nil),
		(*UserBadge)(nil),
	}

	for _, model := range models {
		if _, err := DB.NewCreateTable().IfNotExists().Model(model).Exec(context.Background()); err != nil {
			return err
		}
	}
	return nil
}
