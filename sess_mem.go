package cartsess

import (
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type MemoryStore struct {
	mutex           sync.RWMutex
	Options         *Options               // default configuration
	value           map[string]interface{} //session store
	gc              map[string]int64       //session gc time store
	SessionIDLength int
	GCTime          time.Duration //ever second run GC
}

var _ Store = &MemoryStore{}

func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{
		Options: &Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
		SessionIDLength: 64,
		GCTime:          5 * 60,
		value:           make(map[string]interface{}),
		gc:              make(map[string]int64),
	}
	s.GC()
	return s
}

func (s *MemoryStore) Get(r *http.Request, cookieName string) (session *Session, err error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
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
			session.Values = s.value[sid.Value].(map[string]interface{})
		}
	} else {
		newid := generateID(s.SessionIDLength)
		session.ID = newid
		session.IsNew = true
	}
	return session, err
}

// Save adds a single session to the response.
func (s *MemoryStore) Save(r *http.Request, w http.ResponseWriter, session *Session) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	sid := session.ID
	s.value[sid] = session.Values
	s.gc[sid] = time.Now().Unix()

	cookie := NewCookie(session.CookieName(), session.ID, session.Options)
	http.SetCookie(w, cookie)
	return nil
}

func (s *MemoryStore) Destroy(r *http.Request, w http.ResponseWriter, session *Session) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	sid := session.ID
	delete(s.value, sid)
	delete(s.gc, sid)
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

func (s *MemoryStore) innerGC() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	memage := s.Options.MaxAge
	if memage <= 0 {
		memage = 86400 * 30
	}
	count := 0
	min := time.Now().Unix() - int64(memage)
	for sid := range s.value {
		if s.gc[sid] < min {
			delete(s.value, sid)
			delete(s.gc, sid)
			count++
		}
	}
	if count > 0 {
		log.Printf(infoFormat, "MemoryStore GC romove count:"+strconv.Itoa(count))
	}
	s.GC()
}

func (s *MemoryStore) GC() {
	go func() {
		<-time.After(time.Second * s.GCTime)
		s.innerGC()
	}()
}
