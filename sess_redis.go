package cartsess

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	Options         *Options // default configuration
	SessionIDLength int
	Client          redis.UniversalClient
	Prefix          string
	Serializer      SessionSerializer
}

var _ Store = &RedisStore{}

func Context() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	return ctx, cancel
}

type SessionSerializer interface {
	Deserialize(d []byte, session *Session) error
	Serialize(session *Session) ([]byte, error)
}

type JSONSerializer struct{}

func (s JSONSerializer) Serialize(session *Session) ([]byte, error) {
	m := make(map[string]interface{}, len(session.Values))
	for k, v := range session.Values {
		m[k] = v
	}
	return json.Marshal(m)
}

func (s JSONSerializer) Deserialize(d []byte, session *Session) error {
	m := make(map[string]interface{})
	err := json.Unmarshal(d, &m)
	if err != nil {
		log.Printf("redistore.JSONSerializer.deserialize() Error: %v", err)
		return err
	}
	for k, v := range m {
		session.Values[k] = v
	}
	return nil
}

type GobSerializer struct{}

func (s GobSerializer) Serialize(session *Session) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(session.Values)
	if err == nil {
		return buf.Bytes(), nil
	}
	return nil, err
}

func (s GobSerializer) Deserialize(d []byte, session *Session) error {
	dec := gob.NewDecoder(bytes.NewBuffer(d))
	return dec.Decode(&session.Values)
}

func NewRedisStore(opts ...*redis.Options) *RedisStore {
	var redisOpt *redis.Options
	if len(opts) > 0 {
		redisOpt = opts[0]
	} else {
		redisOpt = &redis.Options{
			Addr:     "localhost:6379",
			Password: "", // no password set
			DB:       0,  // use default DB
		}
	}

	return NewRedisStoreWithClient(redis.NewClient(redisOpt))
}

func NewRedisStoreWithClient(client redis.UniversalClient) *RedisStore {
	return &RedisStore{
		Options: &Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
		SessionIDLength: 64,
		Prefix:          "",
		Client:          client,
		Serializer:      GobSerializer{},
	}
}

func (s *RedisStore) SetSerializer(sessionSerializer SessionSerializer) {
	s.Serializer = sessionSerializer
}

func (s *RedisStore) Get(r *http.Request, cookieName string) (session *Session, err error) {
	session, err = s.New(r, cookieName)
	session.cookieName = cookieName
	session.store = s
	return
}

func (s *RedisStore) New(r *http.Request, cookieName string) (*Session, error) {
	session := NewSession(s, cookieName)
	opts := *s.Options
	session.Options = &opts
	session.IsNew = true
	var err error
	if sid, errCookie := r.Cookie(cookieName); errCookie == nil {
		session.ID = sid.Value
		//get value
		ctx, cancel := Context()
		defer cancel()
		val, _err := s.Client.Get(ctx, s.Prefix+sid.Value).Result()
		if _err == nil {
			_err = s.Serializer.Deserialize([]byte(val), session)
			if _err != nil {
				err = _err
			}
			session.IsNew = false
		} else {
			if _err == redis.Nil {
				err = ErrNotFound
			} else {
				err = _err
			}
			newid := generateID(s.SessionIDLength)
			session.ID = newid
			session.IsNew = true
		}
	} else {
		newid := generateID(s.SessionIDLength)
		session.ID = newid
		session.IsNew = true
	}
	return session, err
}

// Save adds a single session to the response.
func (s *RedisStore) Save(r *http.Request, w http.ResponseWriter, session *Session) error {
	sid := session.ID
	b, err := s.Serializer.Serialize(session)
	if err != nil {
		log.Println(err)
		return err
	}
	ctx, cancel := Context()
	defer cancel()
	err = s.Client.Set(ctx, s.Prefix+sid, string(b), time.Duration(s.Options.MaxAge)*time.Second).Err()
	if err != nil {
		log.Println(err)
		return err
	}

	cookie := NewCookie(session.CookieName(), session.ID, session.Options)
	http.SetCookie(w, cookie)
	return nil
}

func (s *RedisStore) Destroy(r *http.Request, w http.ResponseWriter, session *Session) error {
	sid := session.ID
	ctx, cancel := Context()
	defer cancel()
	s.Client.Del(ctx, s.Prefix+sid)
	opt := &Options{
		Path:     session.Options.Path,
		Domain:   session.Options.Domain,
		Secure:   session.Options.Secure,
		HttpOnly: session.Options.HttpOnly,
		SameSite: session.Options.SameSite,
		MaxAge:   -1,
	}
	http.SetCookie(w, NewCookie(session.CookieName(), "", opt))
	return nil
}
