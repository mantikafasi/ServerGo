package main

import (
	"context"
	"log"
	"server-go/common"
	"server-go/database"
	"server-go/database/schemas"
	"strconv"

	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/state"
)

func main() {
	// common.InitCache()
	// database.InitDB()
	GetUser("1049092894616719400")
	// SendNotification(1)
}

func GetUser(discordid string) {
	ArikawaState, err := state.New("Bot " + common.Config.BotToken)

	if err != nil {
		panic(err)
	}
	discordId, err := strconv.ParseInt(discordid, 10, 64)

	user, err := ArikawaState.User(discord.UserID(discordId))

	if err != nil {
		log.Println(err)
	}

	println(user.Username, " " , user.Discriminator, " ", user.AvatarURL())
}

func SendNotification(userId int32) {
	notification := schemas.Notification{
		UserID:  userId,
		Content: "Hello world \n\n Goodbye World!",
	}

	if _, err := database.DB.NewInsert().Model(&notification).Exec(context.Background()); err != nil {
		log.Println(err)
	}
}
