package cartsess

import (
	"fmt"
	"log"
	"net/http"

	"github.com/teatak/cart/v2"
)

var firstKey string = ""

const (
	prefixKey   = "github.com/teatak/cartsess:"
	errorFormat = "[SESS]  ERROR! %s\n"
	infoFormat  = "[SESS]  INFO %s\n"
)

var ErrNotFound = fmt.Errorf("not found")

// NewSession is called by session stores to create a new session instance.
func NewManager(cookieName string, store Store) cart.Handler {
	return func(c *cart.Context, next cart.Next) {
		s := &SessionManager{
			cookieName: cookieName,
			store:      store,
			request:    c.Request,
			written:    false,
			response:   c.Response,
		}
		c.Set(prefixKey+cookieName, s)
		if firstKey == "" {
			//save firstKey as De
			firstKey = prefixKey + cookieName
		}
		// Register callback to save session before response is written
		c.Response.OnBeforeWrite(func() {
			err := s.Save()
			if err != nil {
				log.Printf(errorFormat, err)
			}
		})

		next()
	}
}

// shortcut to get session
func Default(c *cart.Context) *SessionManager {
	if firstKey == "" {
		panic(fmt.Errorf("must run NewManager before use session"))
	} else {
		return c.MustGet(firstKey).(*SessionManager)
	}
}

func GetByName(c *cart.Context, cookieName string) *SessionManager {
	return c.MustGet(prefixKey + cookieName).(*SessionManager)
}

type SessionManager struct {
	cookieName string
	store      Store
	request    *http.Request
	response   http.ResponseWriter
	session    *Session
	written    bool
}

func (s *SessionManager) Get(key string) (interface{}, error) {
	sess, err := s.Session()
	return sess.Values[key], err
}

func (s *SessionManager) Set(key string, val interface{}) error {
	sess, err := s.Session()
	if sess != nil {
		sess.Values[key] = val
		s.written = true
	}
	return err
}

func (s *SessionManager) Delete(key string) error {
	sess, err := s.Session()
	if sess != nil {
		delete(sess.Values, key)
		s.written = true
	}
	return err
}

func (s *SessionManager) Destroy() error {
	sess, err := s.Session()
	if sess != nil {
		sess.Values = make(map[string]interface{})
		err = sess.Destroy(s.request, s.response)
		//end written
		s.written = false
	}
	return err
}
func (s *SessionManager) Session() (*Session, error) {
	var err error
	if s.session == nil {
		s.session, err = s.store.Get(s.request, s.cookieName)
		if err != nil {
			log.Printf(errorFormat, err)
		}
	}
	return s.session, err
}

// Save is a convenience method to save this session. It is the same as calling
// store.Save(request, response, session). You should call Save before writing to
// the response or returning from the handler.
func (s *SessionManager) Save() error {
	if s.Written() {
		sess, err := s.Session()
		if err == nil {
			err = sess.Save(s.request, s.response)
		}
		//end written
		s.written = false
		return err
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
