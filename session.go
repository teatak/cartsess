package cartsess

import (
	"net/http"
	"github.com/gimke/cart"
)

const (
	DefaultKey  = "github.com/gimke/cartsess"
)

type Options struct {
	Path   string
	Domain string
	MaxAge   int
	Secure   bool
	HttpOnly bool
}

type Session struct {
	name    string
	store   Store
	id		string
	values  map[interface{}]interface{}
	Options *Options
	IsNew   bool
}

// NewSession is called by session stores to create a new session instance.
func NewSession(name string, store Store) cart.Handler {
	return func(c *cart.Context, next cart.Next) {
		s := &Session{
			name:   name,
			store:  store,
			values: make(map[interface{}]interface{}),
		}
		c.Set(DefaultKey,s)
		next()
	}
}

// shortcut to get session
func Default(c *cart.Context) *Session {
	return c.MustGet(DefaultKey).(*Session)
}


// Save is a convenience method to save this session. It is the same as calling
// store.Save(request, response, session). You should call Save before writing to
// the response or returning from the handler.
func (s *Session) Save(r *http.Request, w http.ResponseWriter) error {
	return s.store.Save(r, w, s)
}

// Name returns the name used to register the session.
func (s *Session) Name() string {
	return s.name
}

// Store returns the session store used to register the session.
func (s *Session) Store() Store {
	return s.store
}
