package common

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var PublicKeyString string = "a6953fb61ec1e9107fae66ff1c56437c322f561e2f1578b0f52dbcb3d9eda694"

func VerifySignature(signatureString string, message []byte) bool {
	signature, _ := hex.DecodeString(signatureString)
	publicKey, _ := hex.DecodeString(PublicKeyString)
	return ed25519.Verify(publicKey, message, signature)
}

func SendStructResponse(w http.ResponseWriter, response any) {
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func Ternary[T any](b bool, ifTrue, ifFalse T) T {
	if b {
		return ifTrue
	}

	return ifFalse
}

var urlRegex *regexp.Regexp = regexp.MustCompile(`[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`)
var reviewURLRegex *regexp.Regexp = regexp.MustCompile(`https://[^\s<]+`)

func ContainsURL(s string) bool {
	return urlRegex.MatchString(s)
}

func ContainsOnlyAllowedReviewGIFURLs(s string) bool {
	urls := reviewURLRegex.FindAllString(s, -1)
	if len(urls) == 0 {
		return false
	}

	for _, rawURL := range urls {
		if !isAllowedReviewGIFURL(rawURL) {
			return false
		}
	}

	return true
}

func isGIFProvider(hostname string) bool {
	return hostname == "tenor.com" || strings.HasSuffix(hostname, ".tenor.com") || hostname == "tenor.co" || strings.HasSuffix(hostname, ".tenor.co") || hostname == "giphy.com" || strings.HasSuffix(hostname, ".giphy.com") || hostname == "klipy.com" || strings.HasSuffix(hostname, ".klipy.com") || hostname == "klipy.co" || strings.HasSuffix(hostname, ".klipy.co")
}

func isAllowedReviewGIFURL(rawURL string) bool {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Scheme != "https" || !isGIFProvider(strings.ToLower(parsed.Hostname())) {
		return false
	}

	path := strings.ToLower(parsed.Path)
	return isDirectReviewGIFPath(path)
}

func isDirectReviewGIFPath(path string) bool {
	path = strings.ToLower(path)
	return strings.HasSuffix(path, ".gif") || strings.HasSuffix(path, ".webp") || strings.HasSuffix(path, ".mp4") || strings.HasSuffix(path, "/mp4")
}

const (
	ADDED               = "Added your review"
	ERROR               = "An error occurred"
	UPDATED             = "Updated your review"
	EDITED              = "Successfully updated review"
	DELETED             = "Successfully deleted review"
	UNAUTHORIZED        = "Unauthorized"
	INVALID_TOKEN       = "Invalid Token, please reauthenticate from settings"
	OPTED_OUT           = "You have opted out of ReviewDB"
	INVALID_REVIEW      = "Invalid review"
	INVALID_REVIEW_TYPE = "Invalid review type"
	UPDATE_FAILED       = "An Error occurred while updating your review"
)

type Translate struct {
	Sentences []struct {
		Trans string `json:"trans"`
	} `json:"sentences"`
	Src        string  `json:"src"`
	Confidence float32 `json:"confidence"`
}

func FormatUser(username string, id int32, discordId string) string {
	if id == 0 {
		return fmt.Sprintf("Username: %v\nDiscord ID: %v (<@%v>)", username, discordId, discordId)
	}
	return fmt.Sprintf("Username: %v\nDiscord ID: %v (<@%v>)\nReviewDB ID: %v", username, discordId, discordId, id)
}

func GetQueryOrDefault(r *http.Request, key string, defaultValue string) string {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func GetIntQueryOrDefault(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
