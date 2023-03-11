package redisstore

import (
	"net/http"
	"net/http/httptest"
	"testing"

	gomock "github.com/golang/mock/gomock"
)

func TestStore_Save(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	client := NewMockRedisClient(mockCtrl)

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

}
