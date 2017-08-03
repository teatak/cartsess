package cartsess

import (
	"github.com/gorilla/securecookie"
	"net/http"
)

type CookieStore struct {
	Codecs  []securecookie.Codec
	Options *Options // default configuration
}

var _ Store = &CookieStore{}

func NewCookieStore(keyPairs ...[]byte) *CookieStore {
	cs := &CookieStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
	}

	cs.MaxAge(cs.Options.MaxAge)
	return cs
}

func (s *CookieStore) Get(r *http.Request, cookieName string) (session *Session, err error) {
	session, err = s.New(r, cookieName)
	session.cookieName = cookieName
	session.store = s
	return
}

func (s *CookieStore) New(r *http.Request, name string) (*Session, error) {
	session := NewSession(s, name)
	opts := *s.Options
	session.Options = &opts
	session.IsNew = true
	var err error
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.values,
			s.Codecs...)
		if err == nil {
			session.IsNew = false
		}
	}
	return session, err
}

// Save adds a single session to the response.
func (s *CookieStore) Save(r *http.Request, w http.ResponseWriter,
	session *Session) error {
	encoded, err := securecookie.EncodeMulti(session.CookieName(), session.values,
		s.Codecs...)
	if err != nil {
		return err
	}
	http.SetCookie(w, NewCookie(session.CookieName(), encoded, session.Options))
	return nil
}

func (s *CookieStore) MaxAge(age int) {
	s.Options.MaxAge = age

	// Set the maxAge for each securecookie instance.
	for _, codec := range s.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}