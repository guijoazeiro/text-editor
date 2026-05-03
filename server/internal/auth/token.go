package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
