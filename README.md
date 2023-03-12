# redisstore

[![Go Report Card](https://goreportcard.com/badge/github.com/joelrose/redisstore)](https://goreportcard.com/report/github.com/joelrose/redisstore)
[![codecov](https://codecov.io/gh/joelrose/redisstore/branch/main/graph/badge.svg?token=S7OK5EE8L3)](https://codecov.io/gh/joelrose/redisstore)
[![Tests](https://github.com/joelrose/redisstore/actions/workflows/ci.yml/badge.svg)](https://github.com/joelrose/redisstore/actions/workflows/ci.yaml)
[![GoDoc](https://godoc.org/github.com/joelrose/redisstore?status.svg)](https://godoc.org/github.com/joelrose/redisstore)

A gorilla/sessions store implementation with Redis. 
I was frustrated with the available packages since they are either outdated or too closely linked with a Redis client package. So, I made my own package that is adaptable, modern and well tested.

## Install

```bash
go get github.com/joelrose/redisstore
```

## Example

```go
package main

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/joelrose/redisstore"
	"github.com/joelrose/redisstore/adapter"
	goredis "github.com/redis/go-redis/v9"
    redigo "github.com/gomodule/redigo/redis"
)

func main() {
	// Create a new go-redis client, pool or cluster
	goRedisClient := goredis.NewClient(&goredis.Options{
		Addr: "localhost:6379",
	})

	// Create a new redigo pool
	// redigoPool := &redigo.Pool{
	// 	Dial: func() (redigo.Conn, error) {
	// 		return redigo.Dial("tcp", "localhost:6379")
	// 	},
	// }

	// New Store
	keys := [][]byte{[]byte("hash")}
	store := redisstore.New(
		adapter.UseGoRedis(goRedisClient),
		// adapter.UseRedigo(redigoPool),
		keys,
		redisstore.WithSessionOptions(sessions.Options{
			Path:   "/",
			Domain: "example.com",
			MaxAge: 86400 * 30,
		}),
		redisstore.WithKeyPrefix("prefix_"),
	)

	// Get Session from a http.Request
	var req *http.Request
	session, err := store.Get(req, "session-name")

	// Add your data to the session
	session.Values["foo"] = "bar"

	// Save session to http.ResponseWriter
	var w http.ResponseWriter
	err = sessions.Save(req, w)

	// Delete session (MaxAge <= 0) and save to http.ResponseWriter
	session.Options.MaxAge = -1
	err = sessions.Save(req, w)
}
```

## License

This project is licensed under the MIT license. See the [LICENSE](./LICENSE) file for more
details.