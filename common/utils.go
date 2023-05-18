package common

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

var PublicKeyString string = "a6953fb61ec1e9107fae66ff1c56437c322f561e2f1578b0f52dbcb3d9eda694"

func VerifySignature(signatureString string, message []byte) bool {
	signature,_ := hex.DecodeString(signatureString)
	publicKey, _ := hex.DecodeString(PublicKeyString)
	return ed25519.Verify(publicKey, message, signature)
}

func SendStructResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func Ternary[T any](b bool, ifTrue, ifFalse T) T {
	if b {
		return ifTrue
	}
	return ifFalse
}