package schemas

import (
	"time"

	"github.com/uptrace/bun"
)

type TwitterUser struct {
	bun.BaseModel `bun:"table:reviewdb_twitter.users"`

	ID           int32       `bun:"id,pk,autoincrement" json:"Id"`
	TwitterID    string      `bun:"twitter_id,type:numeric" json:"twitterId"`
	Token        string      `bun:"token" json:"-"`
	Username     string      `bun:"username" json:"username"`
	DisplayName  string      `bun:"display_name" json:"displayName"`
	Type         int32       `bun:"column:type" json:"-"`
	AvatarURL    string      `bun:"avatar_url" json:"avatar_url"`
	WarningCount int32       `bun:"warning_count" json:"warningCount"`
	Badges       []UserBadge `bun:"-" json:"badges"`
	OptedOut     bool        `bun:"opted_out" json:"-"`
	IpHash       string      `bun:"ip_hash" json:"-"`
	RefreshToken string      `bun:"refresh_token" json:"-"`
	ExpiresAt    time.Time   `bun:"expires_at" json:"-"`

	BanID int32 `bun:"ban_id" json:"-"`

	BanInfo *ReviewDBBanLog `bun:"rel:has-one,join:ban_id=id" json:"banInfo"`
}

type ReviewDBTwitterBanLog struct {
	bun.BaseModel `bun:"table:reviewdb_twitter.user_bans"`

	ID              int32     `bun:"id,pk,autoincrement" json:"id"`
	TwitterID       string    `bun:"twitter_id,type:numeric" json:"twitterId"`
	ReviewID        int32     `bun:"review_id" json:"reviewId"`
	ReviewContent   string    `bun:"review_content" json:"reviewContent"`
	AdminDiscordID  *string   `bun:"admin_discord_id,type:numeric" json:"-"`
	BanEndDate      time.Time `bun:"ban_end_date" json:"banEndDate"`
	Timestamp       time.Time `bun:"timestamp,default:current_timestamp" json:"-"`
	ReviewTimestamp time.Time `bun:"review_timestamp" json:"reviewTimestamp"`
}

type TwitterUserReview struct {
	bun.BaseModel `bun:"table:reviewdb_twitter.reviews"`

	ID           int32         `bun:"id,pk,autoincrement" json:"id"`
	ProfileID    string        `bun:"profile_id,type:numeric" json:"-"`
	Sender       TwitterSender `bun:"-" json:"sender"`
	Comment      string        `bun:"comment" json:"comment"`
	Type         int32         `bun:"type" json:"type"` // 0 = normal review , 1 = system review
	TimestampStr time.Time     `bun:"timestamp,default:current_timestamp" json:"-"`
	Timestamp    int64         `bun:"-" json:"timestamp"`

	User       *TwitterUser `bun:"rel:belongs-to,join:reviewer_id=id" json:"-"`
	ReviewerID string       `bun:"reviewer_id,type:numeric" json:"-"`
}

type TwitterSender struct {
	ID        int32              `json:"id"`
	TwitterID string             `json:"twitterId"`
	Username  string             `json:"username"`
	AvatarURL string             `json:"avatarURL"`
	Badges    []TwitterUserBadge `json:"badges"`
}

type TwitterUserBadge struct {
	bun.BaseModel `bun:"table:reviewdb_twitter.user_badges"`

	ID              int32  `bun:"id,pk,autoincrement" json:"-"`
	TargetTwitterID string `bun:"target_twitter_id,type:numeric" json:"-"`
	Name            string `bun:"name" json:"name"`
	Icon            string `bun:"icon_url" json:"icon"`
	RedirectURL     string `bun:"redirect_url" json:"redirectURL"`
	Type            int32  `bun:"type" json:"type"`
	Description     string `bun:"description" json:"description"`
}

type TwitterRequestData struct {
	Comment   string `json:"comment"`
	ProfileID string `json:"profileId"`
}
