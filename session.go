package cartsess

import (
	"net/http"
	"time"
)

type Session struct {
	// The ID of the session, generated by stores. It should not be used for
	// user data.
	ID string
	// Values contains the user-data for the session.
	Values     map[string]interface{}
	Options    *Options
	IsNew      bool
	store      Store
	cookieName string
	TTL        time.Duration
}

func (s *Session) Save(r *http.Request, w http.ResponseWriter) error {
	return s.store.Save(r, w, s)
}

func (s *Session) Destroy(r *http.Request, w http.ResponseWriter) error {
	return s.store.Destroy(r, w, s)
}

func NewSession(store Store, cookieName string) *Session {
	return &Session{
		Values:     make(map[string]interface{}),
		store:      store,
		cookieName: cookieName,
	}
}

func (s *Session) CookieName() string {
	return s.cookieName
}
