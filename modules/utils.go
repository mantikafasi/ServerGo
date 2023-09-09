package modules

import (
	"crypto/rand"
	"encoding/base64"
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
