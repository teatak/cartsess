package cartsess

import (
	"github.com/gorilla/securecookie"
	"net/http"
	"fmt"
)

type MemoryStore struct {
	Codecs  []securecookie.Codec
	Options *Options // default configuration
}

var _ Store = &MemoryStore{}

func NewMemoryStore(keyPairs ...[]byte) *MemoryStore {
	ms := &MemoryStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
	}

	ms.MaxAge(ms.Options.MaxAge)
	return ms
}

func (s *MemoryStore) Get(r *http.Request, cookieName string) (session *Session, err error) {
	session, err = s.New(r, cookieName)
	session.cookieName = cookieName
	session.store = s
	return
}

func (s *MemoryStore) New(r *http.Request, name string) (*Session, error) {
	session := NewSession(s, name)
	opts := *s.Options
	session.Options = &opts
	session.IsNew = true
	var err error
	if c, errCookie := r.Cookie(name); errCookie == nil {
		fmt.Println(c)
		err = securecookie.DecodeMulti(name, c.Value, &session.values,
			s.Codecs...)
		if err == nil {
			session.IsNew = false
		}
	}
	return session, err
}

// Save adds a single session to the response.
func (s *MemoryStore) Save(r *http.Request, w http.ResponseWriter,
	session *Session) error {
	encoded, err := securecookie.EncodeMulti(session.CookieName(), session.values,
		s.Codecs...)
	if err != nil {
		return err
	}
	http.SetCookie(w, NewCookie(session.CookieName(), encoded, session.Options))
	return nil
}

func (s *MemoryStore) MaxAge(age int) {
	s.Options.MaxAge = age

	// Set the maxAge for each securecookie instance.
	for _, codec := range s.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}