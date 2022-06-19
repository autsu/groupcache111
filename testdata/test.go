package main

import (
	"cache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

var (
	port = flag.String("p", "", "port")
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

var getFunc = cache.GetterFunc(
	func(key string) ([]byte, error) {
		if v, ok := db[key]; ok {
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s not exist", key)
	})

func newGroup(name string, size int64, fn cache.Getter) *cache.Group {
	return cache.NewGroup(name, size, fn)
}

// 一个缓存服务器
func startCacheServer(host, port string, peersAddr []string, g *cache.Group) error {
	pool := cache.NewHTTPPool(host, port)
	pool.Set(peersAddr...)
	g.RegisterPeers(pool)
	if err := http.ListenAndServe(fmt.Sprintf("%v:%v", host, port), pool); err != nil {
		return err
	}
	return nil
}

// 该接口用于获取缓存
func startApiServer(host, port string, g *cache.Group) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		view, err := g.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(view.ByteSlice())
	})
	if err := http.ListenAndServe(fmt.Sprintf("%v:%v", host, port), nil); err != nil {
		return err
	}
	return nil
}

func main() {
	//go func() {
	//	host, port := "localhost", "10000"
	//	if err := startApiServer(host, port, group); err != nil {
	//		log.Fatal(err)
	//	}
	//	log.Printf("[api server] listen in: %v\n", host+port)
	//}()
	flag.Parse()

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
	g := newGroup("user_cache", 100, getFunc)
	adr, ok := cacheServerAddr[*port]
	if !ok {
		log.Fatalln("input wrong port, must be one of 10001, 10002, 10003")
	}

	log.Printf("[api server] listen in: %v\n", adr.host+":"+adr.port)
	if err := startCacheServer(adr.host, adr.port, peersAddr, g); err != nil {
		log.Fatalf("[%v] err: %v", adr.host+":"+adr.port, err)
	}
}
