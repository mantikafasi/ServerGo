package modules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/uptrace/bun"
	"server-go/common"
)

type StupitStat struct {
	bun.BaseModel `bun:"table:stupit_table,"`

	ID		int32 `bun:"id,pk,autoincrement"`
	DiscordID int64 `bun:"discordid,"`
	Stupidity int32 `bun:"stupidity,"`
	SenderID  int64 `bun:"senderdiscordid,"`
}

type UserInfoStr struct {
	bun.BaseModel `bun:"table:user_info,"`

	ID        int32  `bun:"id,pk,autoincrement"`
	DiscordID int64  `bun:"discordid,"`
	Token     string `bun:"token,"`
}

func CalculateHash(token string) string {
	checksum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(checksum[:])
}

func AddStupidityDBUser(code string) (string,error) {
	DB := common.GetDB()
	token, err := ExchangeCodePlus(code,"http://192.168.1.35/auth")
	if err != nil {
		return "",err
	}

	discordUser,err :=GetUser(token)

	exists, _ := DB.NewSelect().Where("discordid = ?", discordUser.ID).Model(&UserInfoStr{}).Exists(context.Background())

	var user UserInfoStr
	user.DiscordID = discordUser.ID
	user.Token = CalculateHash(token)

	if exists {
		_, err := DB.NewUpdate().Where("discordid = ?", discordUser.ID).Model(&user).Exec(context.Background())
		if err != nil {
			return "", err
		} else {
			return token, nil
		}
	} else {
		_, err := DB.NewInsert().Model(&user).Exec(context.Background())
		if err != nil {
			return "", err
		}
		return token, nil

	}

}

func GetDiscordIDWithToken( token string) int64 {
	DB := common.GetDB()

	var user UserInfoStr
	err := DB.NewSelect().Where("token = ?", CalculateHash(token)).Model(&user).Scan(context.Background())
	if err != nil {
		return 0
	}
	return user.DiscordID
}

func VoteStupidity( discordID int64, token string, stupidity int32) string {
	DB := common.GetDB()
	senderID := GetDiscordIDWithToken(token)
	
	exists, err := DB.NewSelect().Where("discordid = ?", discordID).Where("senderdiscordid=?", senderID).Model(&StupitStat{}).Exists(context.Background())
	if err != nil {
		return "An Error Occured"
	}

	if exists {
		//update data
		_, err = DB.NewUpdate().Where("discordid = ?", discordID).Model(&StupitStat{Stupidity: stupidity}).Exec(context.Background())
		if err != nil {
			return "An error occured"
		} else {
			return "Updated Your Vote"
		}
	} else {
		var stupit StupitStat
		stupit.DiscordID = discordID
		stupit.Stupidity = stupidity
		stupit.SenderID = senderID
		_, err := DB.NewInsert().Model(&stupit).Exec(context.Background())
		if err != nil {
			return "An Error Occured"
		}
		return "Successfully voted"
	}

}

func GetStupidity(discordID int64) (int, error) {
	DB := common.GetDB()

	//check if user has votes
	exists, err := DB.NewSelect().Where("discordid = ?", discordID).Model(&StupitStat{}).Exists(context.Background())
	if err != nil {
		return -1, err
	}
	if !exists {
		return -1, nil
	}

	var stupidity float64
	
	rows, err := DB.Query("SELECT AVG(stupidity) FROM stupit_table WHERE discordid = ?", discordID)

	DB.ScanRows(context.Background(), rows, &stupidity)

	return int(stupidity), nil

}
