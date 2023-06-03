package groupcache

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"void.io/x/cache/consistenthash"
	"void.io/x/cache/pb/cachepb"

	"google.golang.org/protobuf/proto"
)

const defaultUrl = "/groupcache/"

// DefaultReplicas 默认虚拟节点数量
const DefaultReplicas = 50

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
	host, port string
	// 该节点的地址，格式为："ip|host:port", e.g. "localhost:8080"
	addr  string
	mu    sync.RWMutex        // 保护 peers 和 httpGetters
	peers *consistenthash.Map // 哈希环，用来保存所有节点，同时实现负载均衡

	// 映射远程节点与对应的 httpGetter。每一个远程节点对应一个 httpGetter
	httpGetters map[string]PeerGetter

	// 配置参数，如果不指定，则使用默认值
	baseURL  string                  // /<baseURL>/<groupName>/<key>
	replicas int64                   // hash 环的虚拟节点数
	hashFunc consistenthash.HashFunc // 调用者自定义的哈希函数
}

func NewHTTPPool(host, port string, opts ...HTTPPoolOption) *HTTPPool {
	h := &HTTPPool{
		addr:        fmt.Sprintf("%v:%v", host, port),
		httpGetters: make(map[string]PeerGetter),
	}

	for _, opt := range opts {
		opt(h)
	}

	if h.baseURL == "" {
		h.baseURL = defaultUrl
	}
	if h.replicas == 0 {
		h.replicas = DefaultReplicas
	}
	// 如果 hashFunc 为 nil，那么 New 内部会使用默认的哈希函数
	h.peers = consistenthash.New(h.replicas, h.hashFunc)

	return h
}

func (h *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, h.baseURL) {
		errmsg := fmt.Sprintf(
			`request url need contain a baseURL: %v, 
					e.g <scheme>://<host>/%v/<groupName>/<key>,
					you request url is %v`, h.baseURL, h.baseURL, r.URL.Path)
		w.Write([]byte(errmsg))
		panic(errmsg)
	}
	log.Printf("[%v][%v] %v \n", h.addr, r.Method, r.URL.Path)
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

	// 调用了 group.Get ，如果缓存不存在，则会从数据源获取
	val, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// octet-stream 表示未知的文件类型
	w.Header().Set("Content-Type", "application/octet-stream")
	// 使用 proto 编码响应内容
	resp, err := proto.Marshal(&cachepb.Response{Value: val.ByteSlice()})
	if err != nil {
		log.Println("proto marshal error: ", err)
	}
	if _, err := w.Write(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *HTTPPool) PickPeer(key string) (addr string, peer PeerGetter, notSelf bool) {
	p := h.peers.Get(key)
	// 找到了节点且该节点不是当前节点（如果是当前节点，那么就没必要进行 http 调用去远程获取了，
	// 直接在本地查询即可）
	if p != "" && p != h.addr {
		//log.Printf("[%v] -> Redirected to key[%v] at %v\n", h.addr, key, p)
		// 在其他节点（机器），获得该节点的 Get 方法，进行 http 调用获取结果
		return p, h.httpGetters[p], true
	}
	return p, nil, false
}

func (h *HTTPPool) Addr() string {
	return h.addr
}

func (h *HTTPPool) Set(peers ...string) {
	if h.peers == nil {
		h.peers = consistenthash.New(h.replicas, h.hashFunc)
	}
	h.peers.Add(peers...)
	if h.httpGetters == nil {
		h.httpGetters = make(map[string]PeerGetter)
	}
	for _, peer := range peers {
		h.httpGetters[peer] = &httpGetter{host: peer}
	}
}

// 默认请求 url 格式为：<scheme>://<host>/<baseURL>/<groupName>/<key>
type httpGetter struct {
	scheme  string // http or https
	host    string
	baseURL string
}

func (h *httpGetter) Get(in *cachepb.Request, out *cachepb.Response) error {
	if h.scheme == "" {
		h.scheme = "http"
	}
	if h.host == "" {
		panic("httpGetter error: host is empty")
	}
	if h.baseURL == "" {
		h.baseURL = defaultUrl
	}
	// 因为 path.Join 不能适用于 URL 的格式，所以只能拼接 scheme 后面的部分
	// （path.Join 会把 scheme://a/b 变为 scheme:/a/b）
	p := path.Join(h.host,
		h.baseURL,
		url.QueryEscape(in.Group),
		url.QueryEscape(in.Key))
	// 因为 URL 的形式是 scheme://p，所以 p 不能以 '/' 开头，不然就成了 scheme:///p
	if p[0] == '/' {
		p = p[1:]
	}
	// ps: go1.19 将会在 net/url 添加一个有用的函数 JoinPath 来解决上面的问题
	u := fmt.Sprintf("%v://%v", h.scheme, p)
	res, err := http.Get(u)
	if err != nil {
		log.Println("http get error: ", err)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("server returned: %v", res.Status)
		return fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("reading response body: %v", err)
		return fmt.Errorf("reading response body: %v", err)
	}

	if err := proto.Unmarshal(bytes, out); err != nil {
		log.Println("proto unmarshal error: ", err)
		return err
	}

	return nil
}

var _ PeerGetter = (*httpGetter)(nil)
