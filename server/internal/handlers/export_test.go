package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ParseUserUUID(c *gin.Context) (uuid.UUID, bool) {
	return parseUserUUID(c)
}

func ContainsInsensitive(s, sub string) bool {
	return containsInsensitive(s, sub)
}

func ToLower(r rune) rune {
	return toLower(r)
}
