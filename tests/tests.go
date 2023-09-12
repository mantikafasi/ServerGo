package main

import (
	"context"
	"log"
	"server-go/common"
	"server-go/database"
	"server-go/database/schemas"
)

func main() {
	common.InitCache()
	database.InitDB()

	SendNotification(1)
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
