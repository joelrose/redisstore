package adapter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/joelrose/redisstore"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

type storeFactory func(*testing.T) *redisstore.Store

func TestGetSet_GoRedis(t *testing.T) {
	GetSet(t, func(_ *testing.T) *redisstore.Store {
		client := goredis.NewClient(&goredis.Options{
			Addr: "localhost:6379",
		})

		return redisstore.New(UseGoRedis(client), [][]byte{[]byte("secret")})
	})
}

func TestGetSet_Redigo(t *testing.T) {
	GetSet(t, func(_ *testing.T) *redisstore.Store {
		pool := &redis.Pool{
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", "localhost:6379")
			},
		}

		return redisstore.New(UseRedigo(pool), [][]byte{[]byte("secret")})
	})
}

func GetSet(t *testing.T, newStore storeFactory) {
	const (
		sessionName = "session"
		value       = "ok"
	)
	store := newStore(t)

	set := func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, sessionName)

		session.Values["key"] = value
		err := session.Save(r, w)
		assert.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(value))
	}

	get := func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, sessionName)

		val, ok := session.Values["key"].(string)
		assert.Equal(t, true, ok)
		assert.Equal(t, value, val)

		err := session.Save(r, w)
		assert.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(value))
	}

	delete := func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, sessionName)

		session.Options.MaxAge = -1
		err := session.Save(r, w)
		assert.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(value))
	}

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/set", nil)
	set(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/get", nil)
	copyCookies(req2, res1)
	get(res2, req2)

	res3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/delete", nil)
	copyCookies(req3, res2)
	delete(res3, req3)

	cookies := res3.Result().Cookies()
	assert.Equal(t, 1, len(cookies))
	assert.Equal(t, -1, cookies[0].MaxAge)
	assert.Equal(t, "", cookies[0].Value)
}

func copyCookies(req *http.Request, res *httptest.ResponseRecorder) {
	req.Header.Set("Cookie", strings.Join(res.Header().Values("Set-Cookie"), "; "))
}
