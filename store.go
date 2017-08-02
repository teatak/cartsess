package cartsess

import "net/http"

type Store interface {
	Get(r *http.Request, name string) (*Session, error)
	New(r *http.Request, name string) (*Session, error)
	Save(r *http.Request, w http.ResponseWriter, s *Session) error
	Options(Options)
}