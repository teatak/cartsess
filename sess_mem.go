package cartsess

import (
	"net/http"
	"math/rand"
)

type MemoryStore struct {
	Options 		*Options // default configuration
	value        	map[interface{}]interface{} //session store
	gc        		map[interface{}]interface{} //session store
	SessionIDLength	int
}

var _ Store = &MemoryStore{}

func (s *MemoryStore) GC() {

}

func NewMemoryStore() *MemoryStore {
	ms := &MemoryStore{
		Options: &Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
		SessionIDLength: 64,
		value: make(map[interface{}]interface{}),
	}
	ms.GC()
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
			session.Values = s.value[sid.Value].(map[interface{}]interface{})
		}
	} else {
		newid := s.generateID()
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
	sid := session.ID
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

func (s *MemoryStore) Destroy(r *http.Request, w http.ResponseWriter, session *Session) error {
	sid := session.ID
	delete(s.value,sid)
	delete(s.gc,sid)
	opt := session.Options
	opt.MaxAge = -1
	http.SetCookie(w, NewCookie(session.CookieName(), "", opt))
	return nil
}


var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-")

func (s *MemoryStore) generateID() string {
	b := make([]rune, s.SessionIDLength)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)

}
