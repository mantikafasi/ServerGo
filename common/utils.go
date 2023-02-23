package common

import (
	"crypto/ed25519"
	"encoding/hex"
)

var PublicKeyString string = "a6953fb61ec1e9107fae66ff1c56437c322f561e2f1578b0f52dbcb3d9eda694"

func VerifySignature(signatureString string, message []byte) bool {
	signature,_ := hex.DecodeString(signatureString)
	publicKey, _ := hex.DecodeString(PublicKeyString)
	return ed25519.Verify(publicKey, message, signature)
}
