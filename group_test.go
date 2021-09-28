package cache

import (
	"log"
	"testing"
)

var data = map[string]string{
	"a": "1",
	"b": "2",
	"c": "3",
}

func TestGroup(t *testing.T) {
	group := NewGroup("name", 5*MB, GetterFunc(func(key string) ([]byte, error) {
		v := data[key]
		return []byte(v), nil
	}))

	group = GetGroup("name")

	val, err := group.Get("a")
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(val)

	val, err = group.Get("a")
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(val)

	// Output:
	// 2021/09/28 14:11:50 1
	// 2021/09/28 14:11:50 cache is hit
	// 2021/09/28 14:11:50 1
}
