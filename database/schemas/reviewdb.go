package schemas

import (
	"time"

	"github.com/uptrace/bun"
)

type URUser struct {
	bun.BaseModel `bun:"table:users"`

	ID           int32       `bun:"id,pk,autoincrement" json:"ID"`
	DiscordID    string      `bun:"discord_id,type:numeric" json:"discordID"`
	Token        string      `bun:"token" json:"-"`
	Username     string      `bun:"username" json:"username"`
	Type         int32       `bun:"column:type" json:"-"`
	AvatarURL    string      `bun:"avatar_url" json:"profilePhoto"`
	ClientMods   []string    `bun:"client_mods,array" json:"clientMods"`
	WarningCount int32       `bun:"warning_count" json:"warningCount"`
	Badges       []UserBadge `bun:"-" json:"badges"`
	OptedOut     bool        `bun:"opted_out" json:"-"`
	IpHash       string      `bun:"ip_hash" json:"-"`
	RefreshToken string      `bun:"refresh_token" json:"-"`

	BanID int32 `bun:"ban_id" json:"-"`

	BanInfo *ReviewDBBanLog `bun:"rel:has-one,join:ban_id=id" json:"banInfo"`
}

type AdminUser struct {
	bun.BaseModel `bun:"table:users"`
	DiscordID     string `bun:"discord_id,type:numeric"`
	ProfilePhoto  string `bun:"avatar_url"`
}

type ReviewReport struct {
	bun.BaseModel `bun:"table:reports"`

	ID         int32 `bun:"id,pk,autoincrement"`
	ReviewID   int32 `bun:"review_id"`
	ReporterID int32 `bun:"reporter_id"`
}

type ReviewDBBanLog struct {
	bun.BaseModel `bun:"table:user_bans"`

	ID              int32     `bun:"id,pk,autoincrement" json:"id"`
	DiscordID       string    `bun:"discord_id,type:numeric" json:"discordID"`
	ReviewID        int32     `bun:"review_id" json:"reviewID"`
	ReviewContent   string    `bun:"review_content" json:"reviewContent"`
	AdminDiscordID  *string   `bun:"admin_discord_id,type:numeric" json:"-"`
	BanEndDate      time.Time `bun:"ban_end_date" json:"banEndDate"`
	Timestamp       time.Time `bun:"timestamp,default:current_timestamp" json:"-"`
	ReviewTimestamp time.Time `bun:"review_timestamp" json:"reviewTimestamp"`
}

type ActionLog struct {
	bun.BaseModel `bun:"table:action_log"`

	Action string `bun:"action" json:"action"`

	ReviewID     int32  `bun:"id,pk,autoincrement" json:"id"`
	UserID       int64  `bun:"user_id,type:numeric" json:"-"`
	SenderUserID int32  `bun:"sender_user_id" json:"senderuserid"`
	Comment      string `bun:"comment" json:"comment"`

	UpdatedString string `bun:"comment_new"`
	ActionUserID  int32  `bun:"action_user_id"`
}

type ReviewDBAppeal struct {
	bun.BaseModel `bun:"table:appeals"`

	ID         int32  `bun:"id,pk,autoincrement" json:"id"`
	UserID     int32  `bun:"user_id,type:numeric" json:"-"`
	BanID      int32  `bun:"ban_id" json:"-"`
	AppealText string `bun:"appeal_text" json:"appealText"`
}

type Sender struct {
	ID           int32       `json:"id"`
	DiscordID    string      `json:"discordID"`
	Username     string      `json:"username"`
	ProfilePhoto string      `json:"profilePhoto"`
	Badges       []UserBadge `json:"badges"`
}

type UserReview struct {
	bun.BaseModel `bun:"table:reviews"`

	ID           int32     `bun:"id,pk,autoincrement" json:"id"`
	ProfileID    int64     `bun:"profile_id,type:numeric" json:"-"`
	Sender       Sender    `bun:"-" json:"sender"`
	Comment      string    `bun:"comment" json:"comment"`
	Type         int32     `bun:"type" json:"type"` // 0 = user review , 1 = server review , 2 = support review, 3 = system review
	TimestampStr time.Time `bun:"timestamp,default:current_timestamp" json:"-"`
	Timestamp    int64     `bun:"-" json:"timestamp"`

	User       *URUser `bun:"rel:belongs-to,join:reviewer_id=id" json:"-"`
	ReviewerID int32   `bun:"reviewer_id" json:"-"`
}


type UserBadge struct {
	bun.BaseModel `bun:"table:user_badges"`

	ID              int32  `bun:"id,pk,autoincrement" json:"-"`
	TargetDiscordID string `bun:"target_discord_id,type:numeric" json:"-"`
	Name            string `bun:"name" json:"name"`
	Icon            string `bun:"icon_url" json:"icon"`
	RedirectURL     string `bun:"redirect_url" json:"redirectURL"`
	Type            int32  `bun:"type" json:"type"`
	Description     string `bun:"description" json:"description"`
}

func (user *URUser) IsAdmin() bool {
	return user.Type == 1
}

func (user *URUser) IsBanned() bool {
	if user.Type == -1 {
		return false
	}
	if user.BanInfo == nil {
		return false
	}
	return true
}
