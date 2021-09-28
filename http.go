package cache

import (
	"fmt"
	"go-LRU/consistenthash"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const defaultUrl = "/cache/"

type HTTPPoolOption func(pool *HTTPPool)

func WithBaseURL(url string) HTTPPoolOption {
	return func(pool *HTTPPool) {
		pool.baseURL = url
	}
}

// WithReplicas 指定 hash 环的虚拟节点数
func WithReplicas(replicas int64) HTTPPoolOption {
	return func(pool *HTTPPool) {
		pool.replicas = replicas
	}
}

// WithHashFunc 指定 hash 环所使用的 hash 函数
func WithHashFunc(fn consistenthash.HashFunc) HTTPPoolOption {
	return func(pool *HTTPPool) {
		pool.hashFunc = fn
	}
}

// HTTPPool 保存了当前分布式系统里的所有节点，同时其本身也是一个节点
type HTTPPool struct {
	// 该节点的地址，格式为："ip/host:port", e.g. "localhost:8080"
	addr  string
	mu    sync.RWMutex             // 保护 peers 和 httpGetters
	peers *consistenthash.HashRing // 哈希环，用来保存所有节点，同时实现负载均衡

	// 映射远程节点与对应的 httpGetter。每一个远程节点对应一个 httpGetter
	httpGetters map[string]PeerGetter

	// 配置参数，如果不指定，则使用默认值
	baseURL  string                  // /<baseURL>/<groupName>/<key>
	replicas int64                   // hash 环的虚拟节点数
	hashFunc consistenthash.HashFunc // 调用者自定义的哈希函数
}

func NewHTTPPool(addr string, opts ...HTTPPoolOption) *HTTPPool {
	h := &HTTPPool{
		addr:    addr,
		httpGetters: make(map[string]PeerGetter),
	}

	for _, opt := range opts {
		opt(h)
	}

	if h.baseURL == "" {
		h.baseURL = defaultUrl
	}
	if h.replicas == 0 {
		h.replicas = consistenthash.DefaultReplicas
	}
	// 如果 hashFunc 为 nil，那么 New 内部会使用默认的哈希函数
	h.peers = consistenthash.New(h.replicas, h.hashFunc)

	return h
}

func (h *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, h.baseURL) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	log.Printf("[%v] %v \n", r.Method, r.URL.Path)

	// /<baseURL>/<groupName>/<key>，将 <groupName>/<key> 这部分以 '/' 做为
	// 分隔符，分隔出两个子串，也就是 groupName 和 key
	n := strings.SplitN(r.URL.Path[len(h.baseURL):], "/", 2)
	//log.Println(r.URL.Path, r.URL.Path[len(h.baseURL):])
	if len(n) < 2 {
		http.Error(w, "url format is wrong", http.StatusBadRequest)
		return
	}

	groupName := n[0]
	key := n[1]
	//log.Printf("groupname: %v, key: %v \n", groupName, key)

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// 调用了 group.Get ，如果缓存不存在，则会查找数据源
	val, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// octet-stream 表示未知的文件类型
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := w.Write(val.ByteSlice()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *HTTPPool) PickPeer(key string) (peer PeerGetter, ok bool) {
	p := h.peers.Get(key)
	// 找到了节点且该节点不是当前节点（如果是当前节点，那么就没必要进行 http 调用去远程获取了，
	// 直接在本地查询即可）
	if p != "" && p != h.addr {
		// 在其他节点（机器），获得该节点的 Get 方法，进行 http 调用获取结果
		return h.httpGetters[p], true
	}
	return nil, false
}

type httpGetter struct {
	baseURL string
}

func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

var _ PeerGetter = (*httpGetter)(nil)
