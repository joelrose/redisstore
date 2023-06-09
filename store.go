//go:generate mockgen -source=store.go -destination=store_mock.go -package=redisstore_test
package redisstore

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/rs/xid"
)

type Client interface {
	// Get returns the value for a given key.
	Get(ctx context.Context, key string) ([]byte, error)
	// Set sets the value for a given key.
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	// Del deletes a given key.
	Del(ctx context.Context, key string) error
}

// KeyGenFunc defines a function used by store to generate the session key.
type KeyGenFunc func() string

type Store struct {
	Options    *sessions.Options
	Codecs     []securecookie.Codec
	client     Client
	serializer SessionSerializer
	keyGen     KeyGenFunc
	keyPrefix  string
}

var _ sessions.Store = (*Store)(nil)

type Options func(s *Store)

// WithKeyPrefix sets the key prefix used in redis keys.
func WithKeyPrefix(prefix string) Options {
	return func(s *Store) {
		s.keyPrefix = prefix
	}
}

// WithSerializer sets the serializer used to serialize the session.
// By default, the GobSerializer is used.
func WithSerializer(serializer SessionSerializer) Options {
	return func(s *Store) {
		s.serializer = serializer
	}
}

// WithKeyGenerator sets the key generator used to generate the session key.
// By default, the defaultKeyGenerator method is used.
func WithKeyGenerator(keyGen KeyGenFunc) Options {
	return func(s *Store) {
		s.keyGen = keyGen
	}
}

// WithSessionOptions sets the session options.
func WithSessionOptions(options sessions.Options) Options {
	return func(s *Store) {
		s.Options = &options
	}
}

const (
	defaultMaxAge    = 86400 * 30
	defaultPath      = "/"
	defaultKeyPrefix = "session_"
)

func New(client Client, keyPairs [][]byte, options ...Options) *Store {
	s := &Store{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{ //nolint: exhaustruct
			Path:   defaultPath,
			MaxAge: defaultMaxAge,
		},
		client:     client,
		keyPrefix:  defaultKeyPrefix,
		keyGen:     defaultKeyGenerator,
		serializer: GobSerializer{},
	}

	for _, option := range options {
		option(s)
	}

	s.SetMaxAge(s.Options.MaxAge)

	return s
}

// SetMaxAge sets the maximum age for the store and the underlying cookie
// implementation. Individual sessions can be deleted by setting Options.MaxAge
// = -1 for that session.
//
// ref: https://github.com/gorilla/sessions/blob/0e1d1d7c382124033b710ef1ef0993327195ed40/store.go#L243
func (s *Store) SetMaxAge(age int) {
	s.Options.MaxAge = age

	// Set the maxAge for each securecookie instance.
	for _, codec := range s.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}

// SetOptions sets the options for the store.
func (s *Store) SetOptions(options sessions.Options) {
	s.Options = &options
}

// Get returns a session for the given name after adding it to the registry.
//
// ref: https://github.com/gorilla/sessions/blob/0e1d1d7c382124033b710ef1ef0993327195ed40/store.go#L178
func (s *Store) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name) //nolint: wrapcheck
}

// New returns a session for the given name without adding it to the registry.
//
// ref: https://github.com/gorilla/sessions/blob/0e1d1d7c382124033b710ef1ef0993327195ed40/store.go#L185
func (s *Store) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(s, name)
	options := *s.Options
	session.Options = &options
	session.IsNew = true

	c, errCookie := r.Cookie(name)
	if errCookie != nil {
		return session, nil //nolint: nilerr
	}

	if err := securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...); err != nil {
		return session, fmt.Errorf("redisstore(new): decoding cookie value: %v", err)
	}

	if err := s.load(r.Context(), session); err == nil {
		session.IsNew = false
	}

	return session, nil
}

// Save adds a single session to the response.
//
// If the Options.MaxAge of the session is <= 0, the session is deleted.
func (s *Store) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	// Delete session if max-age is <= 0
	if session.Options.MaxAge <= 0 {
		// TODO(joelrose): find a better solution, not sure if we should use the request context here
		if err := s.delete(r.Context(), session); err != nil {
			return fmt.Errorf("redisstore(save): deleting session: %v", err)
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))

		return nil
	}

	if session.ID == "" {
		session.ID = s.keyGen()
	}

	if err := s.save(r.Context(), session); err != nil {
		return fmt.Errorf("redisstore(save): saving session: %v", err)
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
	if err != nil {
		return fmt.Errorf("redisstore(save): encoding cookie value: %v", err)
	}

	http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))

	return nil
}

// save stores the session in redis.
func (s *Store) save(ctx context.Context, session *sessions.Session) error {
	b, err := s.serializer.Serialize(session)
	if err != nil {
		return fmt.Errorf("serializing session: %v", err)
	}

	maxAge := time.Duration(session.Options.MaxAge) * time.Second
	key := s.keyPrefix + session.ID
	if err := s.client.Set(ctx, key, b, maxAge); err != nil {
		return fmt.Errorf("setting session: %v", err)
	}

	return nil
}

// load reads the session from redis.
func (s *Store) load(ctx context.Context, session *sessions.Session) error {
	val, err := s.client.Get(ctx, s.keyPrefix+session.ID)
	if err != nil {
		return fmt.Errorf("getting session: %v", err)
	}

	if err := s.serializer.Deserialize(val, session); err != nil {
		return fmt.Errorf("deserializing session: %v", err)
	}

	return nil
}

// delete removes session from redis.
func (s *Store) delete(ctx context.Context, session *sessions.Session) error {
	if err := s.client.Del(ctx, s.keyPrefix+session.ID); err != nil {
		return fmt.Errorf("deleting session: %v", err)
	}

	return nil
}

// defaultKeyGenerator generates a new session ID.
func defaultKeyGenerator() string {
	return xid.New().String()
}
