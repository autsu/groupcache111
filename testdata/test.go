package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"void.io/x/cache"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

var (
	port = flag.String("p", "", "port")
	db   = map[string]string{
		"Tom":  "630",
		"Jack": "589",
		"Sam":  "567",
	}
	getFunc = groupcache.GetterFunc(
		func(key string) ([]byte, error) {
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("key[%s] is not exist in db", key)
		})
)

func newGroup(name string, size int64, fn groupcache.Getter) *groupcache.Group {
	return groupcache.NewGroup(name, size, fn)
}

// 一个缓存服务器
func startCacheServer(host, port string, peersAddr []string, g *groupcache.Group) error {
	pool := groupcache.NewHTTPPool(host, port)
	pool.Set(peersAddr...)
	g.RegisterPeers(pool)
	if err := http.ListenAndServe(fmt.Sprintf("%v:%v", host, port), pool); err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	if *port == "" {
		panic("usage: ./test -p [10001 | 10002 | 10003]")
	}
	groupName := "user_cache"
	groupSize := int64(100)

	type addr struct {
		host string
		port string
	}
	cacheServerAddr := make(map[string]*addr)
	cacheServerAddr["10001"] = &addr{"127.0.0.1", "10001"}
	cacheServerAddr["10002"] = &addr{"127.0.0.1", "10002"}
	cacheServerAddr["10003"] = &addr{"127.0.0.1", "10003"}

	var peersAddr []string
	for _, addr := range cacheServerAddr {
		peersAddr = append(
			peersAddr,
			fmt.Sprintf("%v:%v", addr.host, addr.port))
	}
	g := newGroup(groupName, groupSize, getFunc)
	adr, ok := cacheServerAddr[*port]
	if !ok {
		log.Fatalln("input wrong port, must be one of 10001, 10002, 10003")
	}

	log.Printf("[api server] listen in: %v\n", adr.host+":"+adr.port)
	if err := startCacheServer(adr.host, adr.port, peersAddr, g); err != nil {
		log.Fatalf("[%v] err: %v", adr.host+":"+adr.port, err)
	}
}
