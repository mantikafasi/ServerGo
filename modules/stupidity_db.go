package modules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"

	discord_utils "server-go/modules/discord"
	"server-go/common"
	"server-go/database"
	"server-go/database/schemas"
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
	discordToken, err := discord_utils.ExchangeCode(code, common.Config.Origin+"/auth")
	if err != nil {
		return "", err
	}

	discordUser, err := discord_utils.GetUser(discordToken.AccessToken)
	if err != nil {
		return "", err
	}
	token := GenerateToken()

	var user = &schemas.UserInfo{DiscordID: discordUser.ID, Token: token}

	res, err := database.DB.NewUpdate().Where("discord_id = ?", discordUser.ID).Model(user).Exec(context.Background())
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

	return token, nil
}

func GetDiscordIDWithToken(token string) string {
	var user *schemas.UserInfo
	err := database.DB.NewSelect().Where("token = ?", CalculateHash(token)).Model(user).Scan(context.Background())
	if err != nil {
		return "0"
	}
	return user.DiscordID
}

func VoteStupidity(discordID int64, token string, stupidity int32, senderDiscordID string) string {
	var senderID string
	if token == common.Config.StartItBotToken {
		senderID = senderDiscordID
	} else {
		senderID = GetDiscordIDWithToken(token)
	}

	stupit := &schemas.StupitStat{ReviewedDiscordID: discordID, StupidityValue: stupidity, ReviewerDiscordID: senderID}

	res, err := database.DB.
		NewUpdate().
		Where("reviewed_discord_id = ?", discordID).
		Where("reviewer_discord_id = ?", senderID).
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
		Where("reviewed_discord_id = ?", discordID).
		Model((*schemas.StupitStat)(nil)).
		Exists(context.Background())
	if err != nil || !exists {
		return -1, err
	}

	rows, err := database.DB.Query("SELECT AVG(stupidity_value) FROM stupidity_reviews WHERE reviewed_discord_id = ?", discordID)
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
