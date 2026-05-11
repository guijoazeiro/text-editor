package testutil

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	internalauth "github.com/guijoazeiro/text-editor/tree/main/server/internal/auth"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/config"
)

const TestJWTSecret = "test-secret-key-for-unit-tests"

func NewTestJWT() *internalauth.JWT {
	cfg := &config.Config{JWTSecret: TestJWTSecret}
	return internalauth.NewJWT(cfg)
}

func GenerateTestToken(userID uuid.UUID, email string) (string, error) {
	j := NewTestJWT()
	return j.GenerateToken(userID, email)
}

func GenerateExpiredToken(userID uuid.UUID, email string) (string, error) {
	claims := internalauth.Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(TestJWTSecret))
}
