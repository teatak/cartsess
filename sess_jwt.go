package cartsess

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTStore implements the Store interface using JWT tokens.
// Session data is encoded in the token itself (stateless).
type JWTStore struct {
	SigningKey    []byte            // Secret key for signing tokens
	SigningMethod jwt.SigningMethod // Signing algorithm (default: HS256)
	Options       *Options          // Cookie options
}

var _ Store = &JWTStore{}

// NewJWTStore creates a new JWTStore with the given signing key.
func NewJWTStore(signingKey []byte) *JWTStore {
	return &JWTStore{
		SigningKey:    signingKey,
		SigningMethod: jwt.SigningMethodHS256,
		Options: &Options{
			Path:     "/",
			MaxAge:   86400 * 7, // 7 days
			HttpOnly: true,
		},
	}
}

// NewJWTStoreWithKeyValidation creates a new JWTStore and validates the key length.
func NewJWTStoreWithKeyValidation(signingKey []byte) (*JWTStore, error) {
	if len(signingKey) < 32 {
		return nil, errors.New("signing key must be at least 32 bytes")
	}
	return NewJWTStore(signingKey), nil
}

// Get retrieves a session from the request.
// It checks Authorization header first, then falls back to cookie.
func (s *JWTStore) Get(r *http.Request, name string) (*Session, error) {
	session, err := s.New(r, name)
	session.cookieName = name
	session.store = s
	return session, err
}

// New creates a new session and attempts to load existing data from JWT token.
func (s *JWTStore) New(r *http.Request, name string) (*Session, error) {
	session := NewSession(s, name)
	opts := *s.Options
	session.Options = &opts
	session.IsNew = true

	// Try to get token from request
	tokenString := s.getTokenFromRequest(r, name)
	if tokenString == "" {
		return session, nil
	}

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if token.Method.Alg() != s.SigningMethod.Alg() {
			return nil, errors.New("invalid signing method")
		}
		return s.SigningKey, nil
	})

	if err != nil {
		return session, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Extract session data from claims
		if data, exists := claims["data"].(map[string]interface{}); exists {
			session.Values = data
			session.IsNew = false
		}
	}

	return session, nil
}

// Save encodes the session as a JWT token and sets it as a cookie.
func (s *JWTStore) Save(r *http.Request, w http.ResponseWriter, session *Session) error {
	// Create claims
	now := time.Now()
	claims := jwt.MapClaims{
		"data": session.Values,
		"iat":  now.Unix(),
	}

	// Add expiration if MaxAge is set
	if session.Options.MaxAge > 0 {
		claims["exp"] = now.Add(time.Duration(session.Options.MaxAge) * time.Second).Unix()
	}

	// Create and sign token
	token := jwt.NewWithClaims(s.SigningMethod, claims)
	tokenString, err := token.SignedString(s.SigningKey)
	if err != nil {
		return err
	}

	// Set cookie
	cookie := NewCookie(session.CookieName(), tokenString, session.Options)
	http.SetCookie(w, cookie)

	// Also set token in response header for API clients
	w.Header().Set("X-JWT-Token", tokenString)

	return nil
}

// Destroy removes the session by setting an expired cookie.
func (s *JWTStore) Destroy(r *http.Request, w http.ResponseWriter, session *Session) error {
	opt := &Options{
		Path:     session.Options.Path,
		Domain:   session.Options.Domain,
		Secure:   session.Options.Secure,
		HttpOnly: session.Options.HttpOnly,
		MaxAge:   -1,
	}
	http.SetCookie(w, NewCookie(session.CookieName(), "", opt))
	return nil
}

// MaxAge sets the maximum age for the store's options.
func (s *JWTStore) MaxAge(age int) {
	s.Options.MaxAge = age
}

// getTokenFromRequest extracts JWT token from request.
// Priority: Authorization header > Cookie
func (s *JWTStore) getTokenFromRequest(r *http.Request, cookieName string) string {
	// 1. Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Support "Bearer <token>" format
		if token, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
			return token
		}
		// Also support raw token
		return authHeader
	}

	// 2. Fall back to cookie
	if cookie, err := r.Cookie(cookieName); err == nil {
		return cookie.Value
	}

	return ""
}
