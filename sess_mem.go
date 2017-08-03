package cartsess

import (
	"net/http"
	"math/rand"
	"time"
	"errors"
	"sync"
	"log"
	"strconv"
)

type MemoryStore struct {
	mutex 			sync.RWMutex
	Options 		*Options // default configuration
	value        	map[string]interface{} //session store
	gc        		map[string]int64 //session gc time store
	SessionIDLength	int
	GCTime			time.Duration
}

var _ Store = &MemoryStore{}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{
		Options: &Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
		SessionIDLength: 	64,
		GCTime:				60,
		value: 				make(map[string]interface{}),
		gc: 				make(map[string]int64),
	}
	go s.GC()
	return s
}

func (s *MemoryStore) Get(r *http.Request, cookieName string) (session *Session, err error) {
	s.mutex.RLock()
	session, err = s.New(r, cookieName)
	session.cookieName = cookieName
	session.store = s
	s.mutex.RUnlock()
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
		newid, errId := s.generateID()
		err = errId
		session.ID = newid
		session.IsNew = true
	}

	return session, err
}

// Save adds a single session to the response.
func (s *MemoryStore) Save(r *http.Request, w http.ResponseWriter, session *Session) error {
	s.mutex.Lock()
	sid := session.ID
	first := false
	if s.value[sid] == nil {
		first = true
	}

	s.value[sid] = session.Values
	s.gc[sid] = time.Now().Unix()

	//if not find in mem
	if first {
		http.SetCookie(w, NewCookie(session.CookieName(), session.ID, session.Options))
	} else {
		http.SetCookie(w, NewCookie(session.CookieName(), session.ID, session.Options))
	}
	s.mutex.Unlock()
	return nil
}

func (s *MemoryStore) Destroy(r *http.Request, w http.ResponseWriter, session *Session) error {
	s.mutex.Lock()
	sid := session.ID
	delete(s.value,sid)
	delete(s.gc,sid)
	opt := session.Options
	opt.MaxAge = -1
	http.SetCookie(w, NewCookie(session.CookieName(), "", opt))
	s.mutex.Unlock()
	return nil
}


var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-")

func (s *MemoryStore) generateID() (string,error) {
	var err error = nil
	if s.SessionIDLength < 32 {
		err = errors.New("SessionIDLength is too short the value should >= 32")
		s.SessionIDLength = 32
	}
	b := make([]rune, s.SessionIDLength)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b),err

}

func (s *MemoryStore) GC() {
	s.mutex.Lock()
	count := 0
	min := time.Now().Unix() - int64(s.Options.MaxAge)
	for sid := range s.value {
		if s.gc[sid] < min {
			s.clear(sid)
			count++
		}
	}
	if count > 0 {
		log.Printf(errorInfo,"MemoryStore GC count:"+strconv.Itoa(count))
	}
	s.mutex.Unlock()

	select {
		case <- time.After(time.Second * s.GCTime):
		go s.GC()
	}
}

func (s *MemoryStore) clear(sid string) {
	delete(s.value, sid)
	delete(s.gc, sid)
}