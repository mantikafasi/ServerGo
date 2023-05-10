package database

import (
	"context"
	"time"

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

type Sender struct {
	ID           int32       `json:"id"`
	DiscordID    string      `json:"discordID"`
	Username     string      `json:"username"`
	ProfilePhoto string      `json:"profilePhoto"`
	Badges       []UserBadge `json:"badges"`
}

type UserReview struct {
	bun.BaseModel `bun:"table:userreviews"`

	ID           int32     `bun:"id,pk,autoincrement" json:"id"`
	UserID       int64     `bun:"userid,type:numeric" json:"-"`
	Sender       Sender    `bun:"-" json:"sender"`
	Star         int32     `bun:"star" json:"star"`
	Comment      string    `bun:"comment" json:"comment"`
	ReviewType   int32     `bun:"reviewtype" json:"type"` // 0 = user review , 1 = server review , 2 = support review, 3 = system review
	TimestampStr time.Time `bun:"timestamp,default:current_timestamp" json:"-"`
	Timestamp    int64     `bun:"-" json:"timestamp"`

	User         *URUser `bun:"rel:belongs-to,join:senderuserid=id" json:"-"`
	SenderUserID int32   `bun:"senderuserid" json:"-"`
}

type UserBadge struct {
	bun.BaseModel `bun:"table:userbadges"`

	ID               int32  `bun:"id,pk,autoincrement" json:"-"`
	DiscordID        string `bun:"discordid,type:numeric" json:"-"`
	BadgeName        string `bun:"badge_name" json:"name"`
	BadgeIcon        string `bun:"badge_icon" json:"icon"`
	RedirectURL      string `bun:"redirect_url" json:"redirectURL"`
	BadgeType        int32  `bun:"badge_type" json:"type"`
	BadgeDescription string `bun:"badge_description" json:"description"`
}

type URUser struct {
	bun.BaseModel `bun:"table:ur_users"`

	ID           int32       `bun:"id,pk,autoincrement" json:"ID"`
	DiscordID    string      `bun:"discordid,type:numeric" json:"discordID"`
	Token        string      `bun:"token" json:"-"`
	Username     string      `bun:"username" json:"username"`
	UserType     int32       `bun:"column:type" json:"-"`
	ProfilePhoto string      `bun:"profile_photo" json:"profilePhoto"`
	ClientMod    string      `bun:"client_mod" json:"clientMod"`
	WarningCount int32       `bun:"warning_count" json:"warningCount"`
	BanEndDate   time.Time   `bun:"ban_end_date" json:"banEndDate"`
	Badges       []UserBadge `bun:"-" json:"badges"`
	OptedOut     bool        `bun:"opted_out" json:"-"`
	IpHash       string      `bun:"ip_hash" json:"-"`
	BanID        int32       `bun:"ban_id" json:"-"`
}

type AdminUser struct {
	bun.BaseModel `bun:"table:ur_users"`
	DiscordID     string `bun:"discordid,type:numeric"`
	ProfilePhoto  string `bun:"profile_photo"`
}

type ReviewReport struct {
	bun.BaseModel `bun:"table:ur_reports"`

	ID         int32 `bun:"id,pk,autoincrement"`
	UserID     int32 `bun:"userid"`
	ReviewID   int32 `bun:"reviewid"`
	ReporterID int32 `bun:"reporterid"`
}

type ReviewDBBanLog struct {
	bun.BaseModel `bun:"table:reviewdb_bans"`

	ID             int32     `bun:"id,pk,autoincrement"`
	DiscordID      string    `bun:"discord_id"`
	ReviewID       int32     `bun:"review_id"`
	ReviewContent  string    `bun:"review_content"`
	AdminDiscordID string    `bun:"admin_discord_id"`
	BanEndDate     time.Time `bun:"ban_end_date"`
	Timestamp      time.Time `bun:"timestamp,default:current_timestamp"`
}

type ActionLog struct {
	bun.BaseModel `bun:"table:actionlog"`

	Action string `bun:"action" json:"action"`

	ReviewID        int32  `bun:"id,pk,autoincrement" json:"id"`
	UserID          int64  `bun:"userid,type:numeric" json:"-"`
	SenderUserID    int32  `bun:"senderuserid" json:"senderuserid"`
	Comment         string `bun:"comment" json:"comment"`

	UpdatedString string `bun:"updatedstring"`
	ActionUserID  int32  `bun:"actionuserid"`
}

func createSchema() error {
	models := []any{
		(*StupitStat)(nil),
		(*UserInfo)(nil),
		(*UserReview)(nil),
		(*URUser)(nil),
		(*ReviewReport)(nil),
		(*ActionLog)(nil),
		(*ReviewDBBanLog)(nil),
	}

	for _, model := range models {
		if _, err := DB.NewCreateTable().IfNotExists().Model(model).Exec(context.Background()); err != nil {
			return err
		}
	}
	return nil
}
