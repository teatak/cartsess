package cartsess

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
)

var (
	firstKey     string
	firstKeyOnce sync.Once
)

const (
	prefixKey   = "github.com/teatak/cartsess:"
	errorFormat = "[SESS]  ERROR! %s\n"
	infoFormat  = "[SESS]  INFO %s\n"
)

var ErrNotFound = fmt.Errorf("not found")

// writerWrapper wraps http.ResponseWriter to intercept WriteHeader
// and save the session before headers are written.
type writerWrapper struct {
	http.ResponseWriter
	sess        *SessionManager
	wroteHeader bool
}

func (w *writerWrapper) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *writerWrapper) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	// Save session before writing headers
	err := w.sess.Save()
	if err != nil {
		log.Printf(errorFormat, err)
	}
	w.ResponseWriter.WriteHeader(code)
}

// NewManager creates a standard net/http middleware for session management.
func NewManager(cookieName string, store Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := &SessionManager{
				cookieName: cookieName,
				store:      store,
				request:    r,
				written:    false,
				response:   w,
			}

			// Store session manager in context
			ctx := context.WithValue(r.Context(), prefixKey+cookieName, s)

			firstKeyOnce.Do(func() {
				firstKey = prefixKey + cookieName
			})
			// Also store as first key if it's the first one initialized (best effort for context)
			// Note: This relies on global state which is tricky with context.
			// Ideally users should use context keys properly.
			// For backward compatibility with "Default", we rely on the global firstKey
			// but we need to ensure the value is in the context with that firstKey too?
			// Actually, "Default" uses "firstKey" to look up. So if we save with firstKey, it works.
			if firstKey == prefixKey+cookieName {
				// Optimization: if this IS the first key, no need to duplicate?
				// But we need to make sure subsequent lookups work.
				// The previous implementation used c.Set(prefixKey+cookieName) AND logic for firstKey.
				// Here we just use the unique key.
			}

			// Wrap response writer to handle session saving
			wrapper := &writerWrapper{
				ResponseWriter: w,
				sess:           s,
			}

			// We update the request with new context
			// Check if firstKey is different from current key
			if firstKey != "" && firstKey != prefixKey+cookieName {
				// If we have a firstKey defined globally, we might want to ensure it's accessible?
				// But context is per-request. firstKey is global.
				// If multiple middleware are used, they chain.
				// If this IS the middleware for firstKey, it's already set.
			}

			next.ServeHTTP(wrapper, r.WithContext(ctx))
		})
	}
}

// Default gets the default session manager from the context.
func Default(ctx context.Context) *SessionManager {
	if firstKey == "" {
		panic(fmt.Errorf("must run NewManager before use session"))
	}
	// Try to get from context
	if v := ctx.Value(firstKey); v != nil {
		return v.(*SessionManager)
	}
	// Fallback or panic? "Default" implies it must exist.
	// In standard net/http, if middleware didn't run, this will panic or return nil.
	panic(fmt.Errorf("session not found in context (did you wrap the handler with NewManager?)"))
}

// GetByName gets a named session manager from the context.
func GetByName(ctx context.Context, cookieName string) *SessionManager {
	if v := ctx.Value(prefixKey + cookieName); v != nil {
		return v.(*SessionManager)
	}
	panic(fmt.Errorf("session '%s' not found in context", cookieName))
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
	if sess == nil {
		return nil, err
	}
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
