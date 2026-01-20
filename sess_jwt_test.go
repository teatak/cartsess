package cartsess

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTStore_NewSession(t *testing.T) {
	store := NewJWTStore([]byte("test-secret-key"))
	req := httptest.NewRequest("GET", "/", nil)

	session, err := store.Get(req, "jwt-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !session.IsNew {
		t.Error("expected session to be new")
	}

	if session.CookieName() != "jwt-session" {
		t.Errorf("expected cookie name 'jwt-session', got '%s'", session.CookieName())
	}
}

func TestJWTStore_SaveAndLoad(t *testing.T) {
	store := NewJWTStore([]byte("test-secret-key"))

	// Create and save session
	req1 := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	session, _ := store.Get(req1, "jwt-session")
	session.Values["user_id"] = float64(123) // JSON numbers are float64
	session.Values["username"] = "testuser"

	err := session.Save(req1, rec)
	if err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	// Check cookie was set
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}

	// Check X-JWT-Token header
	tokenHeader := rec.Header().Get("X-JWT-Token")
	if tokenHeader == "" {
		t.Error("expected X-JWT-Token header to be set")
	}

	// Load session from cookie
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(cookies[0])

	loadedSession, err := store.Get(req2, "jwt-session")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if loadedSession.IsNew {
		t.Error("expected session not to be new")
	}

	if loadedSession.Values["user_id"] != float64(123) {
		t.Errorf("expected user_id 123, got %v", loadedSession.Values["user_id"])
	}

	if loadedSession.Values["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got %v", loadedSession.Values["username"])
	}
}

func TestJWTStore_LoadFromHeader(t *testing.T) {
	store := NewJWTStore([]byte("test-secret-key"))

	// Create a valid token
	claims := jwt.MapClaims{
		"data": map[string]interface{}{
			"role": "admin",
		},
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("test-secret-key"))

	// Create request with Authorization header
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	session, err := store.Get(req, "jwt-session")
	if err != nil {
		t.Fatalf("failed to load session from header: %v", err)
	}

	if session.IsNew {
		t.Error("expected session not to be new")
	}

	if session.Values["role"] != "admin" {
		t.Errorf("expected role 'admin', got %v", session.Values["role"])
	}
}

func TestJWTStore_Destroy(t *testing.T) {
	store := NewJWTStore([]byte("test-secret-key"))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	session, _ := store.Get(req, "jwt-session")
	err := session.Destroy(req, rec)
	if err != nil {
		t.Fatalf("failed to destroy session: %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}

	if cookies[0].MaxAge != -1 {
		t.Errorf("expected MaxAge -1, got %d", cookies[0].MaxAge)
	}
}

func TestJWTStore_InvalidToken(t *testing.T) {
	store := NewJWTStore([]byte("test-secret-key"))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	session, err := store.Get(req, "jwt-session")
	if err == nil {
		t.Error("expected error for invalid token")
	}

	// Session should still be created but as new
	if !session.IsNew {
		t.Error("expected session to be new for invalid token")
	}
}
