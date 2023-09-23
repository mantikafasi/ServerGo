package modules

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"server-go/database"
	"server-go/database/schemas"
)

func GenerateToken() string {
	b := make([]byte, 64)

	if _, err := rand.Read(b); err != nil {
		return ""
	}
	encoder := base64.StdEncoding.WithPadding(base64.NoPadding)
	token := encoder.EncodeToString(b)

	return "rdb." + token
}

func SendNotification(notification *schemas.Notification) (err error) {
	_, err = database.DB.NewInsert().Model(notification).Exec(context.Background())
	if err != nil {
		println(err.Error())
	}
	return
}

func ReadNotification(user *schemas.URUser, notificationId int32) (err error) {
	res, err := database.DB.NewDelete().Model(&schemas.Notification{}).Where("id = ?", notificationId).Where("user_id = ?", user.ID).Exec(context.Background())
	if err != nil {
		return
	}

	rowsAffected, err := res.RowsAffected()

	if rowsAffected == 0 {
		fmt.Println("Couldnt delete notification")
	}
	return
}
