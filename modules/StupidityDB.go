package modules

//TODO write addUser Function
//test code

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/uptrace/bun"
)

type StupitStat struct {
	bun.BaseModel `bun:"table:stupit_table,"`

	DiscordID int64 `bun:"discordid,"`
	Stupidity int32 `bun:"stupidity,"`
	SenderID  int64 `bun:"senderdiscordid,"`
}

type UserInfoStr struct {
	bun.BaseModel `bun:"table:user_info,"`

	ID        int32  `bun:"id,"`
	DiscordID int64  `bun:"discordid,"`
	Token     string `bun:"token,"`
}

func CalculateHash(token string) string {
	checksum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(checksum[:])
}

func AddStupidityDBUser(DB *bun.DB, discordID int64, token string) (string, error) {
	// check if user exists
	exists, _ := DB.NewSelect().Where("discordid = ?", discordID).Model(&UserInfoStr{}).Exists(context.Background())

	var user UserInfoStr
	user.DiscordID = discordID
	user.Token = CalculateHash(token)

	if exists {
		_, err := DB.NewUpdate().Where("discordid = ?", discordID).Model(&user).Exec(context.Background())
		if err != nil {
			return "", err
		} else {
			return "Updated User", nil
		}
	} else {
		_, err := DB.NewInsert().Model(&user).Exec(context.Background())
		if err != nil {
			return "An Error Occured", err
		}
		return "Successfully added user", nil

	}

}

func GetDiscordIDWithToken(DB *bun.DB, token string) int64 {

	var user UserInfoStr
	err := DB.NewSelect().Where("token = ?", CalculateHash(token)).Model(&user).Scan(context.Background())
	if err != nil {
		return 0
	}
	return user.DiscordID
}

func VoteStupidity(DB *bun.DB, discordID int64, token string, stupidity int32) (string, error) {
	senderID := GetDiscordIDWithToken(DB, token)

	exists, err := DB.NewSelect().Where("discordid = ?", discordID).Where("senderdiscordid=?", senderID).Model(&StupitStat{}).Exists(context.Background())
	if err != nil {
		return "", err
	}

	if exists {
		//update data
		_, err = DB.NewUpdate().Where("discordid = ?", discordID).Model(&StupitStat{Stupidity: stupidity}).Exec(context.Background())
		if err != nil {
			return "", err
		} else {
			return "Updated Your Vote", nil
		}
	} else {
		var stupit StupitStat
		stupit.DiscordID = discordID
		stupit.Stupidity = stupidity
		stupit.SenderID = senderID
		_, err := DB.NewInsert().Model(&stupit).Exec(context.Background())
		if err != nil {
			return "An Error Occured", err
		}
		return "Successfully voted", nil
	}

}

func GetStupidity(DB *bun.DB, discordID int64) (int, error) {
	var stat []StupitStat
	err := DB.NewSelect().Where("discordid = ?", discordID).Model(&stat).Scan(context.Background())
	if err != nil {
		return 0, err
	}
	var stupidity float64

	rows, err := DB.Query("SELECT AVG(stupidity) FROM stupit_table WHERE discordid = ?", discordID)
	DB.ScanRows(context.Background(), rows, &stupidity)

	return int(stupidity), nil

}
