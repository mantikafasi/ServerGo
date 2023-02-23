package common


import ("crypto/ed25519")

var publicKey []byte = []byte("a6953fb61ec1e9107fae66ff1c56437c322f561e2f1578b0f52dbcb3d9eda694")

func VerifySignature(signature []byte,message []byte) bool {
	
	return ed25519.Verify(publicKey, message , signature)
}