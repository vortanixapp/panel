package secretbox

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func TokenHash(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
