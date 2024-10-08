package schemas

import (
	"time"

	"github.com/uptrace/bun"
)

const (
	UserTypeBanned    = -1
	UserTypeUser      = 0
	UserTypeAdmin     = 1
	UserTypeModerator = 2
)

type URUser struct {
	bun.BaseModel `bun:"table:users"`

	ID                int32         `bun:"id,pk,autoincrement" json:"ID"`
	DiscordID         string        `bun:"discord_id,type:numeric" json:"discordID"`
	Token             string        `bun:"token" json:"-"`
	Username          string        `bun:"username" json:"username"`
	Type              int32         `bun:"column:type" json:"-"`
	AvatarURL         string        `bun:"avatar_url" json:"profilePhoto"`
	ClientMods        []string      `bun:"client_mods,array" json:"clientMods"`
	WarningCount      int32         `bun:"warning_count" json:"warningCount"`
	Badges            []UserBadge   `bun:"-" json:"badges"`
	OptedOut          bool          `bun:"opted_out" json:"-"`
	IpHash            string        `bun:"ip_hash" json:"-"`
	RefreshToken      string        `bun:"refresh_token" json:"-"`
	AccessToken       string        `bun:"access_token" json:"-"`
	AccessTokenExpiry time.Time     `bun:"access_token_expiry" json:"-"`
	Notification      *Notification `json:"notification" bun:"rel:has-one,join:id=user_id"`
	BlockedUsers      []string      `bun:"blocked_users,array" json:"blockedUsers"`
	Flags             int32         `bun:"flags" json:"flags"`
	LastOnline        time.Time     `bun:"last_online" json:"-"`

	BanID int32 `bun:"ban_id" json:"-"`

	BanInfo *ReviewDBBanLog `bun:"rel:has-one,join:ban_id=id" json:"banInfo"`
}

type ReviewDBUserFull struct {
	bun.BaseModel `bun:"table:users"`

	ID                int32         `bun:"id,pk,autoincrement" json:"id"`
	DiscordID         string        `bun:"discord_id,type:numeric" json:"discord_id"`
	Token             string        `bun:"token" json:"-"`
	Username          string        `bun:"username" json:"username"`
	Type              int32         `bun:"column:type" json:"type"`
	AvatarURL         string        `bun:"avatar_url" json:"profile_photo"`
	ClientMods        []string      `bun:"client_mods,array" json:"client_mods"`
	WarningCount      int32         `bun:"warning_count" json:"warning_count"`
	Badges            []UserBadge   `bun:"-" json:"badges"`
	OptedOut          bool          `bun:"opted_out" json:"opted_out"`
	IpHash            string        `bun:"ip_hash" json:"ip_hash"`
	RefreshToken      string        `bun:"refresh_token" json:"-"`
	AccessToken       string        `bun:"access_token" json:"-"`
	AccessTokenExpiry time.Time     `bun:"access_token_expiry" json:"-"`
	Notification      *Notification `json:"notification" bun:"rel:has-one,join:id=user_id"`
	BlockedUsers      []string      `bun:"blocked_users,array" json:"blocked_users"`
	Flags             int32         `bun:"flags" json:"flags"`

	BanID int32 `bun:"ban_id" json:"-"`

	BanInfo *ReviewDBBanLog `bun:"rel:has-one,join:ban_id=id" json:"ban_info"`
}

type BaseRDBUser struct {
	bun.BaseModel `bun:"table:users"`

	ID        int32       `bun:"id,pk,autoincrement" json:"ID"`
	DiscordID string      `bun:"discord_id,type:numeric" json:"discordID"`
	Username  string      `bun:"username" json:"username"`
	Type      int32       `bun:"column:type" json:"-"`
	AvatarURL string      `bun:"avatar_url" json:"profilePhoto"`
	Badges    []UserBadge `bun:"-" json:"badges"`
	OptedOut  bool        `bun:"opted_out" json:"-"`
}

type AdminUser struct {
	bun.BaseModel `bun:"table:users"`
	DiscordID     string `bun:"discord_id,type:numeric"`
	ProfilePhoto  string `bun:"avatar_url"`
}

type ReviewReport struct {
	bun.BaseModel `bun:"table:reports"`

	ID         int32            `bun:"id,pk,autoincrement"`
	ReviewID   int32            `bun:"review_id"`
	ReporterID int32            `bun:"reporter_id"`
	Review     UserReviewBasic  `bun:"rel:has-one,join:review_id=id" json:"review"`
	Reporter   ReviewDBUserFull `bun:"rel:has-one,join:reporter_id=id" json:"reporter"`
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

const (
	NotificationTypeInfo = iota
	NotificationTypeBan
	NotificationTypeUnban
	NotificationTypeWarning
)

type NotificationType int32

type Notification struct {
	bun.BaseModel `bun:"table:notifications"`

	ID     int32 `bun:"id,pk,autoincrement" json:"id"`
	UserID int32 `bun:"user_id,type:numeric" json:"-"`

	Type      NotificationType `bun:"type" json:"type"`
	Title     string           `bun:"title" json:"title"`
	Content   string           `bun:"content" json:"content"`
	Read      bool             `bun:"read" json:"read"`
	Timestamp time.Time        `bun:"timestamp,default:current_timestamp" json:"-"`
}

type ReviewDBAppeal struct {
	bun.BaseModel `bun:"table:appeals"`

	ID          int32  `bun:"id,pk,autoincrement" json:"id"`
	UserID      int32  `bun:"user_id,type:numeric" json:"-"`
	BanID       int32  `bun:"ban_id" json:"-"`
	AppealText  string `bun:"appeal_text" json:"appealText"`
	ActionTaken bool   `bun:"action_taken" json:"actionTaken"`
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
	Type         int32     `bun:"type" json:"type"` // 0 = user review , 1 = server review , 2 = support review, 3 = system review, 4 = bot integration review
	TimestampStr time.Time `bun:"timestamp,default:current_timestamp" json:"-"`
	Timestamp    int64     `bun:"-" json:"timestamp"`
	ReviewerID   int32     `bun:"reviewer_id" json:"-"`
	RepliesTo    int32     `bun:"replies_to,nullzero" json:"-"`

	User    *URUser      `bun:"rel:belongs-to,join:reviewer_id=id" json:"-"`
	Replies []UserReview `bun:"-" json:"replies"`
}

type UserReviewBasic struct {
	bun.BaseModel `bun:"table:reviews"`

	ID           int32     `bun:"id,pk,autoincrement" json:"id"`
	ProfileID    int64     `bun:"profile_id,type:numeric" json:"-"`
	Comment      string    `bun:"comment" json:"comment"`
	Type         int32     `bun:"type" json:"type"` // 0 = user review , 1 = server review , 2 = support review, 3 = system review, 4 = bot integration review
	TimestampStr time.Time `bun:"timestamp,default:current_timestamp" json:"-"`
	Timestamp    int64     `bun:"-" json:"timestamp"`
	ReviewerID   int32     `bun:"reviewer_id" json:"reviewer_id"`
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
		return true
	}
	if user.BanInfo == nil {
		return false
	}
	return true
}
