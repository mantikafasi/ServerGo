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
	DiscordID       int64  `json:"discordid"`
	Token           string `json:"token"`
	Stupidity       int32  `json:"stupidity"`
	SenderDiscordID string `json:"senderdiscordid"`
}

func CalculateHash(token string) string {
	checksum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(checksum[:])
}

func AddStupidityDBUser(code string) (string, error) {
	token, err := ExchangeCode(code, common.Config.Origin+"/auth")
	if err != nil {
		return "", err
	}

	discordUser, err := GetUser(token.AccessToken)
	if err != nil {
		return "", err
	}

	var user = &database.UserInfo{DiscordID: discordUser.ID, Token: CalculateHash(token.AccessToken)}

	res, err := database.DB.NewUpdate().Where("discordid = ?", discordUser.ID).Model(user).Exec(context.Background())
	if err != nil {
		return "", err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return "", err
	}

	if rowsAffected == 0 {
		_, err = database.DB.NewInsert().Model(user).Exec(context.Background())
		if err != nil {
			return "", err
		}
	}

	return token.AccessToken, nil
}

func GetDiscordIDWithToken(token string) string {
	var user *database.UserInfo
	err := database.DB.NewSelect().Where("token = ?", CalculateHash(token)).Model(user).Scan(context.Background())
	if err != nil {
		return "0"
	}
	return user.DiscordID
}

func VoteStupidity(discordID int64, token string, stupidity int32, senderDiscordID string) string {
	var senderID string
	if token == common.Config.StupidityBotToken {
		senderID = senderDiscordID
	} else {
		senderID = GetDiscordIDWithToken(token)
	}

	stupit := &database.StupitStat{ReviewedDiscordID: discordID, StupidityValue: stupidity, ReviewerDiscordID: senderID}

	res, err := database.DB.
		NewUpdate().
		Where("discord_id = ?", discordID).
		Where("sender_discord_id = ?", senderID).
		Model(stupit).
		Exec(context.Background())
	if err != nil {
		log.Println(err)
		return "An error occurred"
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return "An error occurred"
	}
	if rowsAffected != 0 {
		return "Updated Your Vote"
	}

	_, err = database.DB.NewInsert().Model(stupit).Exec(context.Background())
	if err != nil {
		return "An Error Occurred"
	}

	return "Successfully voted"
}

func GetStupidity(discordID int64) (int, error) {
	// check if user has votes
	exists, err := database.DB.
		NewSelect().
		Where("discord_id = ?", discordID).
		Model((*database.StupitStat)(nil)).
		Exists(context.Background())
	if err != nil || !exists {
		return -1, err
	}

	rows, err := database.DB.Query("SELECT AVG(stupidity) FROM stupidity_reviews WHERE discord_id = ?", discordID)
	defer func() {
		err := rows.Close()
		if err != nil {
			print("Failed to release Rows connection this may be bad")
		}
	}()

	if err != nil {
		return -1, err
	}

	var stupidity float64 = -1
	err = database.DB.ScanRows(context.Background(), rows, &stupidity)

	if err != nil {
		return -1, err
	}

	return int(stupidity), nil
}
