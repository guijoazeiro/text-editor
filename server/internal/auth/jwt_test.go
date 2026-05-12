package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/auth"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/config"
)

func newJWT(secret string) *auth.JWT {
	return auth.NewJWT(&config.Config{JWTSecret: secret})
}

func TestGenerateToken_ValidClaims(t *testing.T) {
	j := newJWT("supersecret")
	userID := uuid.New()
	email := "alice@example.com"

	tokenStr, err := j.GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	claims, err := j.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken returned error: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected Email %s, got %s", email, claims.Email)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	j1 := newJWT("secret-one")
	j2 := newJWT("secret-two")

	tokenStr, _ := j1.GenerateToken(uuid.New(), "alice@example.com")

	_, err := j2.ValidateToken(tokenStr)
	if err == nil {
		t.Fatal("expected error when validating token with wrong secret, got nil")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	j := newJWT("supersecret")

	claims := auth.Claims{
		UserID: uuid.New(),
		Email:  "expired@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte("supersecret"))

	_, err := j.ValidateToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateToken_MalformedToken(t *testing.T) {
	j := newJWT("supersecret")
	_, err := j.ValidateToken("not.a.valid.jwt.at.all")
	if err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}

func TestValidateToken_EmptyToken(t *testing.T) {
	j := newJWT("supersecret")
	_, err := j.ValidateToken("")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

func TestGenerateRefreshToken_ValidClaims(t *testing.T) {
	j := newJWT("supersecret")
	userID := uuid.New()

	tokenStr, err := j.GenerateRefreshToken(userID)
	if err != nil {
		t.Fatalf("GenerateRefreshToken returned error: %v", err)
	}

	gotID, err := j.ValidateRefreshToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateRefreshToken returned error: %v", err)
	}
	if gotID != userID {
		t.Errorf("expected UserID %s, got %s", userID, gotID)
	}
}

func TestValidateRefreshToken_WrongSecret(t *testing.T) {
	j1 := newJWT("secret-one")
	j2 := newJWT("secret-two")

	tokenStr, _ := j1.GenerateRefreshToken(uuid.New())
	_, err := j2.ValidateRefreshToken(tokenStr)
	if err == nil {
		t.Fatal("expected error when validating refresh token with wrong secret, got nil")
	}
}

func TestValidateRefreshToken_AccessTokenRejected(t *testing.T) {
	j := newJWT("supersecret")
	accessToken, _ := j.GenerateToken(uuid.New(), "alice@example.com")

	_, err := j.ValidateRefreshToken(accessToken)
	if err == nil {
		t.Fatal("expected refresh validation to reject an access token, got nil")
	}
}

func TestNewJWT_DefaultSecret(t *testing.T) {
	j := newJWT("")
	tokenStr, err := j.GenerateToken(uuid.New(), "test@example.com")
	if err != nil {
		t.Fatalf("expected token with default secret, got: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}
}

func TestGenerateToken_HasThreeParts(t *testing.T) {
	j := newJWT("supersecret")
	tokenStr, _ := j.GenerateToken(uuid.New(), "bob@example.com")
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		t.Errorf("JWT should have 3 parts separated by '.', got %d", len(parts))
	}
}
