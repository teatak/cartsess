package cartsess

import (
	"crypto/rand"
	"net/http"
	"sync"
	"fmt"
	"encoding/hex"
)

type MemoryStore struct {
	Options *Options // default configuration
	sid          string                      //session id
	value        map[interface{}]map[interface{}]interface{} //session store
	lock         sync.RWMutex
	SessionIDLength	int
}

var _ Store = &MemoryStore{}

func NewMemoryStore(keyPairs ...[]byte) *MemoryStore {
	ms := &MemoryStore{
		Options: &Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
		SessionIDLength: 16,
		value: make(map[interface{}]map[interface{}]interface{}),
	}
	return ms
}

func (s *MemoryStore) Get(r *http.Request, cookieName string) (session *Session, err error) {
	session, err = s.New(r, cookieName)
	session.cookieName = cookieName
	session.store = s
	return
}

func (s *MemoryStore) New(r *http.Request, cookieName string) (*Session, error) {
	session := NewSession(s, cookieName)
	opts := *s.Options
	session.Options = &opts
	session.IsNew = true
	var err error
	if sid, errCookie := r.Cookie(cookieName); errCookie == nil {
		session.ID = sid.Value
		//get value
		if s.value[sid.Value] != nil {
			session.Values = s.value[sid.Value]
		}
	} else {
		newid,err := s.generateID()
		session.ID = newid
		if err == nil {
			session.IsNew = true
		}
	}
	return session, err
}

// Save adds a single session to the response.
func (s *MemoryStore) Save(r *http.Request, w http.ResponseWriter,
	session *Session) error {
	//encoded, err := securecookie.EncodeMulti(session.CookieName(), session.Values,
	//	s.Codecs...)
	//if err != nil {
	//	return err
	//}
	sid := session.ID
	fmt.Println(session)
	first := false
	if s.value[sid] == nil {
		first = true
		s.value[sid] = session.Values
	} else {
		s.value[sid] = session.Values
	}
	//if not find in mem
	if first {
		http.SetCookie(w, NewCookie(session.CookieName(), session.ID, session.Options))
	}
	return nil
}


func (s *MemoryStore) generateID() (string, error) {
	b := make([]byte, s.SessionIDLength)
	n, err := rand.Read(b)
	if n != len(b) || err != nil {
		return "", fmt.Errorf("Could not successfully read from the system CSPRNG")
	}
	return hex.EncodeToString(b), nil
}