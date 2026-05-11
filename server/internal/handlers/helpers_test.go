package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/handlers"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newGinContext(w *httptest.ResponseRecorder) *gin.Context {
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	return c
}

func TestParseUserUUID_MissingKey(t *testing.T) {
	w := httptest.NewRecorder()
	c := newGinContext(w)
	uid, ok := handlers.ParseUserUUID(c)
	if ok {
		t.Fatal("expected ok=false when user_id is not set")
	}
	if uid != uuid.Nil {
		t.Errorf("expected uuid.Nil on failure, got %s", uid)
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestParseUserUUID_ValidUUIDType(t *testing.T) {
	w := httptest.NewRecorder()
	c := newGinContext(w)
	expected := uuid.New()
	c.Set("user_id", expected)

	uid, ok := handlers.ParseUserUUID(c)
	if !ok {
		t.Fatal("expected ok=true with valid uuid.UUID in context")
	}
	if uid != expected {
		t.Errorf("expected %s, got %s", expected, uid)
	}
}

func TestParseUserUUID_ValidStringType(t *testing.T) {
	w := httptest.NewRecorder()
	c := newGinContext(w)
	expected := uuid.New()
	c.Set("user_id", expected.String())

	uid, ok := handlers.ParseUserUUID(c)
	if !ok {
		t.Fatal("expected ok=true when user_id is a valid UUID string")
	}
	if uid != expected {
		t.Errorf("expected %s, got %s", expected, uid)
	}
}

func TestParseUserUUID_InvalidStringType(t *testing.T) {
	w := httptest.NewRecorder()
	c := newGinContext(w)
	c.Set("user_id", "not-a-valid-uuid")

	_, ok := handlers.ParseUserUUID(c)
	if ok {
		t.Fatal("expected ok=false for invalid UUID string")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestParseUserUUID_WrongType(t *testing.T) {
	w := httptest.NewRecorder()
	c := newGinContext(w)
	c.Set("user_id", 12345)
	_, ok := handlers.ParseUserUUID(c)
	if ok {
		t.Fatal("expected ok=false for wrong type in context")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestContainsInsensitive_BasicMatch(t *testing.T) {
	cases := []struct {
		s, sub string
		want   bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "xyz", false},
		{"", "abc", false},
		{"abc", "", true},
		{"", "", true},
		{"Go Lang", "go", true},
		{"GoLang", "lang", true},
		{"abc", "abcd", false},
	}

	for _, tc := range cases {
		got := handlers.ContainsInsensitive(tc.s, tc.sub)
		if got != tc.want {
			t.Errorf("ContainsInsensitive(%q, %q) = %v, want %v", tc.s, tc.sub, got, tc.want)
		}
	}
}

func TestToLower_ASCIILetters(t *testing.T) {
	cases := []struct {
		input rune
		want  rune
	}{
		{'A', 'a'},
		{'Z', 'z'},
		{'M', 'm'},
		{'a', 'a'},
		{'0', '0'},
		{' ', ' '},
	}

	for _, tc := range cases {
		got := handlers.ToLower(tc.input)
		if got != tc.want {
			t.Errorf("ToLower(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
