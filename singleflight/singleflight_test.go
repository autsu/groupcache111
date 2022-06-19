package singleflight

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var g = &Group{
	m: make(map[string]*call),
}

var db = map[string]int64{
	"TOM":  18,
	"JACK": 20,
	"WANG": 22,
}

var idx int64

func Test(t *testing.T) {
	count := 50
	var wg sync.WaitGroup
	wg.Add(count)
	key1 := "TOM"
	key2 := "JACK"
	for i := 0; i < count; i++ {
		go func(i_ int) {
			defer wg.Done()
			v, err := g.Do(key1, func() (value any, err error) {
				value, ok := db[key1]
				if !ok {
					err = fmt.Errorf("key[%v] not exist", key1)
				}
				if atomic.LoadInt64(&idx) == 0 {
					time.Sleep(time.Second * 5)
				}
				atomic.AddInt64(&idx, 1)
				return
			})
			if err != nil {
				log.Println(err)
			}
			log.Printf("id=%v, value: %v\n", i_, v)
		}(i)
	}
	v, err := g.Do(key2, func() (value any, err error) {
		value, ok := db[key2]
		if !ok {
			err = fmt.Errorf("key[%v] not exist", key2)
		}
		return
	})
	if err != nil {
		log.Println(err)
	}
	log.Printf("id=999, value: %v\n", v)
	wg.Wait()
}
