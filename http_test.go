package cache

import (
	"log"
	"net/http"
	"testing"
)

// curl http://localhost:8080/cache/name/a
func TestHTTP(t *testing.T) {
	NewGroup("name", 5*MB, GetterFunc(func(key string) ([]byte, error) {
		v := data[key]
		return []byte(v), nil
	}))

	pool := NewHTTPPool("localhost:8080")
	if err := http.ListenAndServe("localhost:8080", pool); err != nil {
		log.Fatalln(err)
	}

}
