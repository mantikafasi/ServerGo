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

func ReadNotification(user *schemas.URUser, notificationId int32) (err error) {
	res, err := database.DB.NewUpdate().Model(&schemas.Notification{}).Where("id = ?", notificationId).Where("user_id = ?", user.ID).Set("read = true").Exec(context.Background())
	if err != nil {
		return
	}

	rowsAffected, err := res.RowsAffected()

	if rowsAffected == 0 {
		fmt.Println("Couldnt update notification")
	}
	return
}
