package redisstore

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/joelrose/redisstore/mocks"
	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	t.Run("WithKeyPrefix", func(t *testing.T) {
		store := &Store{}
		WithKeyPrefix("prefix_")(store)
		assert.Equal(t, "prefix_", store.keyPrefix)
	})

	t.Run("WithSerializer_Gob", func(t *testing.T) {
		store := &Store{}
		WithSerializer(GobSerializer{})(store)
		assert.Equal(t, GobSerializer{}, store.serializer)
	})

	t.Run("WithSerializer_JSON", func(t *testing.T) {
		store := &Store{}
		WithSerializer(JSONSerializer{})(store)
		assert.Equal(t, JSONSerializer{}, store.serializer)
	})

	t.Run("WithKeyGenerator", func(t *testing.T) {
		store := &Store{}
		WithKeyGenerator(func() string { return "key" })(store)
		assert.Equal(t, "key", store.keyGen())
	})

	t.Run("WithSessionOptions", func(t *testing.T) {
		options := sessions.Options{
			Path:     "/path",
			MaxAge:   0,
			HttpOnly: true,
			Domain:   "domain",
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		}

		store := &Store{}
		WithSessionOptions(options)(store)

		assert.Equal(t, options, *store.Options)
	})
}

func TestStoreNew(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	client := mocks.NewMockRedisClient(mockCtrl)

	store := New(client, [][]byte{[]byte("hash"), []byte("block")})
	assert.NotNil(t, store)
}

func TestStoreSetMaxAge(t *testing.T) {
	store := Store{
		Options: &sessions.Options{
			MaxAge: 0,
		},
		Codecs: securecookie.CodecsFromPairs([]byte("hash"), []byte("block")),
	}

	const maxAge = 10
	store.SetMaxAge(maxAge)
	assert.Equal(t, maxAge, store.Options.MaxAge)
}

func TestStoreSetOptions(t *testing.T) {
	store := Store{}

	options := sessions.Options{
		Path:     "/path",
		MaxAge:   0,
		HttpOnly: true,
		Domain:   "domain",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}

	store.SetOptions(options)

	assert.Equal(t, options, *store.Options)
}

func TestStoreSave(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	client := mocks.NewMockRedisClient(mockCtrl)

	client.EXPECT().Set(gomock.Any(), "prefix_key", gomock.Any(), gomock.Any()).Return(nil)

	keyPairs := [][]byte{[]byte("key")}
	store := New(
		client,
		keyPairs,
		WithKeyGenerator(func() string {
			return "key"
		}),
		WithKeyPrefix("prefix_"),
	)

	req, err := http.NewRequest(http.MethodGet, "http://www.example.com", nil) //nolint:noctx
	if err != nil {
		t.Fatal("failed to create request", err)
	}
	w := httptest.NewRecorder()

	session, err := store.New(req, "test")
	if err != nil {
		t.Fatal("failed to create session", err)
	}

	session.Values["key"] = "value"
	err = session.Save(req, w)
	if err != nil {
		t.Fatal("failed to save: ", err)
	}

	h := w.Header()
	cookies, ok := h["Set-Cookie"]
	if !ok || len(cookies) != 1 {
		t.Fatal("no cookies in header: ", h)
	}
}

func TestStore_Delete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	client := mocks.NewMockRedisClient(mockCtrl)

	clientSet := client.EXPECT().Set(
		gomock.Any(),
		"prefix_key",
		gomock.Any(),
		gomock.Any(),
	)

	clientSet.Do(func(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
		t.Log("set", key, value, expiration)
		return nil
	})

	client.EXPECT().Del(gomock.Any(), "prefix_key").Return(nil)

	keyPairs := [][]byte{[]byte("key")}
	store := New(
		client,
		keyPairs,
		WithKeyGenerator(func() string {
			return "key"
		}),
		WithKeyPrefix("prefix_"),
	)

	req, err := http.NewRequest(http.MethodGet, "http://www.example.com", nil) //nolint:noctx
	if err != nil {
		t.Fatal("failed to create request", err)
	}
	w := httptest.NewRecorder()

	session, err := store.New(req, "test")
	if err != nil {
		t.Fatal("failed to create session", err)
	}

	session.Values["key"] = "value"
	if err := session.Save(req, w); err != nil {
		t.Fatal("failed to save: ", err)
	}

	h := w.Header()
	cookies, ok := h["Set-Cookie"]
	if !ok || len(cookies) != 1 {
		t.Fatal("no cookies in header: ", h)
	}

	session.Options.MaxAge = -1
	if err := session.Save(req, w); err != nil {
		t.Fatal("failed to delete: ", err)
	}
}

func TestDefaultKeyGenerator(t *testing.T) {
	t.Run("generates a unique key", func(t *testing.T) {
		key1 := defaultKeyGenerator()
		key2 := defaultKeyGenerator()

		assert.NotEqual(t, key1, key2)
	})

	t.Run("generates a key of the correct length", func(t *testing.T) {
		key := defaultKeyGenerator()

		assert.Equal(t, 20, len(key))
	})
}
