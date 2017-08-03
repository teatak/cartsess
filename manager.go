package cartsess

import (
	"net/http"
	"github.com/gimke/cart"
	"log"
)

var firstKey string = ""

const (
	prefixKey  = "github.com/gimke/cartsess:"
	errorFormat = "[sessions] ERROR! %s\n"
)

// NewSession is called by session stores to create a new session instance.
func NewManager(cookieName string, store Store) cart.Handler {
	return func(c *cart.Context, next cart.Next) {
		s := &SessionManager{
			cookieName:   	cookieName,
			store:  	store,
			request: 	c.Request,
			written: 	false,
			response: 	c.Response,
		}
		c.Set(prefixKey + cookieName,s)
		if firstKey == "" {
			//save firstKey as De
			firstKey = prefixKey + cookieName
		}
		next()
	}
}

// shortcut to get session
func Default(c *cart.Context) *SessionManager {
	return c.MustGet(firstKey).(*SessionManager)
}

func GetByName(c *cart.Context, cookieName string) *SessionManager {
	return c.MustGet(prefixKey + cookieName).(*SessionManager)
}

type SessionManager struct {
	cookieName	string
	store   	Store
	id			string
	request 	*http.Request
	response  	http.ResponseWriter
	session		*Session
	written		bool
}

func (s *SessionManager) Get(key interface{}) interface{} {
	return s.Session().values[key]
}

func (s *SessionManager) Set(key interface{}, val interface{}) {
	s.Session().values[key] = val
	s.written = true
	s.Save()
}

func (s *SessionManager) Delete(key interface{}) {
	delete(s.Session().values, key)
	s.written = true
	s.Save()
}

func (s *SessionManager) Session() *Session {
	if s.session == nil {
		var err error
		s.session, err = s.store.Get(s.request, s.cookieName)
		if err != nil {
			log.Printf(errorFormat, err)
		}
	}
	return s.session
}
// Save is a convenience method to save this session. It is the same as calling
// store.Save(request, response, session). You should call Save before writing to
// the response or returning from the handler.
func (s *SessionManager) Save() error {
	if s.Written() {
		e := s.Session().Save(s.request, s.response)
		if e == nil {
			s.written = false
		}
		return e
	}
	return nil
}

// Name returns the name used to register the session.
func (s *SessionManager) Name() string {
	return s.cookieName
}

// Store returns the session store used to register the session.
func (s *SessionManager) Store() Store {
	return s.store
}

func (s *SessionManager) Written() bool {
	return s.written
}