package auth_test

import (
	"testing"

	"github.com/guijoazeiro/text-editor/tree/main/server/internal/auth"
)

func TestHashPassword_DifferentFromInput(t *testing.T) {
	password := "mySecretPassword123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == password {
		t.Error("hash must be different from the original password")
	}
	if hash == "" {
		t.Error("hash must not be empty")
	}
}

func TestHashPassword_DifferentHashesSamePassword(t *testing.T) {
	password := "samePassword"
	hash1, _ := auth.HashPassword(password)
	hash2, _ := auth.HashPassword(password)
	if hash1 == hash2 {
		t.Error("expected different hashes for the same password due to bcrypt salt")
	}
}

func TestCheckPasswordHash_CorrectPassword(t *testing.T) {
	password := "correctPassword"
	hash, _ := auth.HashPassword(password)
	if !auth.CheckPasswordHash(password, hash) {
		t.Error("CheckPasswordHash should return true for the correct password")
	}
}

func TestCheckPasswordHash_WrongPassword(t *testing.T) {
	hash, _ := auth.HashPassword("correctPassword")
	if auth.CheckPasswordHash("wrongPassword", hash) {
		t.Error("CheckPasswordHash should return false for a wrong password")
	}
}

func TestCheckPasswordHash_EmptyPassword(t *testing.T) {
	hash, _ := auth.HashPassword("somePassword")
	if auth.CheckPasswordHash("", hash) {
		t.Error("CheckPasswordHash should return false for an empty password")
	}
}

func TestCheckPasswordHash_EmptyHash(t *testing.T) {
	if auth.CheckPasswordHash("somePassword", "") {
		t.Error("CheckPasswordHash should return false for an empty hash")
	}
}

func TestHashRefreshToken_Deterministic(t *testing.T) {
	token := "some-refresh-token-value"
	h1 := auth.HashRefreshToken(token)
	h2 := auth.HashRefreshToken(token)
	if h1 != h2 {
		t.Error("HashRefreshToken must be deterministic")
	}
	if h1 == token {
		t.Error("HashRefreshToken must not return the original token")
	}
	if h1 == "" {
		t.Error("HashRefreshToken must not return an empty string")
	}
}

func TestHashRefreshToken_DifferentInputs(t *testing.T) {
	h1 := auth.HashRefreshToken("token-a")
	h2 := auth.HashRefreshToken("token-b")
	if h1 == h2 {
		t.Error("different tokens must produce different hashes")
	}
}
