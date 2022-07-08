package modules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"

	"server-go/common"
	"server-go/database"
)

type SDB_RequestData struct {
	DiscordID int64  `json:"discordid"`
	Token     string `json:"token"`
	Stupidity   int32 `json:"stupidity"`
}

func CalculateHash(token string) string {
	checksum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(checksum[:])
}

func AddStupidityDBUser(code string) (string, error) {
	token, err := ExchangeCodePlus(code, common.Config.Origin+"/auth")
	if err != nil {
		return "", err
	}

	discordUser, err := GetUser(token)
	var user database.UserInfo

	exists, _ := database.DB.
		NewSelect().
		Where("discordid = ?", discordUser.ID).
		Model(&user).
		Exists(context.Background())

	user.DiscordID = discordUser.ID
	user.Token = CalculateHash(token)

	if exists {
		_, err = database.DB.NewUpdate().Where("discordid = ?", discordUser.ID).Model(&user).Exec(context.Background())
		if err != nil {
			return "", err
		} else {
			return token, nil
		}
	} else {
		_, err = database.DB.NewInsert().Model(&user).Exec(context.Background())
		if err != nil {
			return "", err
		}
		return token, nil
	}
}

func GetDiscordIDWithToken(token string) string {
	var user database.UserInfo
	err := database.DB.NewSelect().Where("token = ?", CalculateHash(token)).Model(&user).Scan(context.Background())
	if err != nil {
		return "0"
	}
	return user.DiscordID
}

func VoteStupidity(discordID int64, token string, stupidity int32) string {
	senderID := GetDiscordIDWithToken(token)

	exists, err := database.DB.
		NewSelect().
		Where("discordid = ?", discordID).
		Where("senderdiscordid = ?", senderID).
		Model((*database.StupitStat)(nil)).
		Exists(context.Background())
	if err != nil {
		return "An Error Occurred"
	}

	stupit := database.StupitStat{DiscordID: discordID, Stupidity: stupidity, SenderID: senderID}
	if exists {
		// update data
		_, err = database.DB.
			NewUpdate().
			Where("discordid = ?", discordID).
			Where("senderdiscordid = ?", senderID).
			Model(&stupit).
			Exec(context.Background())
		if err != nil {
			log.Println(err)
			return "An error occurred"
		} else {
			return "Updated Your Vote"
		}
	} else {
		_, err = database.DB.NewInsert().Model(&stupit).Exec(context.Background())
		if err != nil {
			return "An Error Occurred"
		}
		return "Successfully voted"
	}
}

func GetStupidity(discordID int64) (int, error) {
	// check if user has votes
	exists, err := database.DB.
		NewSelect().
		Where("discordid = ?", discordID).
		Model((*database.StupitStat)(nil)).
		Exists(context.Background())
	if err != nil || !exists {
		return -1, err
	}

	rows, err := database.DB.Query("SELECT AVG(stupidity) FROM stupit_table WHERE discordid = ?", discordID)
	if err != nil {
		return -1, err
	}
	var stupidity float64
	err = database.DB.ScanRows(context.Background(), rows, &stupidity)
	if err != nil {
		return -1, err
	}

	return int(stupidity), nil
}
