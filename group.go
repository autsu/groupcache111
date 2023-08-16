package groupcache

import (
	"fmt"
	"log"
	"sync"

	"void.io/x/cache/pb/cachepb"
	"void.io/x/cache/singleflight"
)

// Getter 定义了获取缓存的方式，当缓存不存在时，应当从某处获取数据，并添加到缓存中，
// 用户通过实现该接口，来自定义当缓存不存在时，从何处（比如 mysql）获取数据
type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (g GetterFunc) Get(key string) ([]byte, error) {
	return g(key)
}

var (
	groups = make(map[string]*Group)
	mu     sync.RWMutex
)

// Group 是一个缓存的命名空间，不同的 Group 可以提供不同的缓存服务，通过 name 来区分
// 比如如果一个 Group 的 name 是 student，说明这个 Group 提供的是学生的缓存信息
type Group struct {
	name      string // 全局唯一
	getter    Getter // 缓存未命中时获取源数据的回调
	mainCache *cache
	peers     PeerPicker
	loader    singleflight.Group
}

func NewGroup(name string, size int64, getter Getter) *Group {
	if getter == nil {
		panic("getter cannot be nil")
	}

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: &cache{size: size},
	}
	mu.Lock()
	defer mu.Unlock()
	groups[name] = g

	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	return groups[name]
}

func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

func (g *Group) Get(key string) (*ByteView, error) {
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	// 从 lru 中查找
	val, exist := g.mainCache.get(key)
	if exist {
		log.Printf("[%v] groupcache is hit\n", g.peers.Addr())
		return val, nil
	}
	// 缓存中不存在，则去指定的数据源中获取
	return g.load(key)
}

// load 当缓存不在当前节点时调用该方法
func (g *Group) load(key string) (value *ByteView, err error) {
	// 使用 singleflight 进行缓存请求
	v, err := g.loader.Do(key, func() (any, error) {
		// 如果有远程节点，则需要确定这个 key 应该交给哪个节点进行处理（负载均衡）
		if g.peers != nil {
			// 确定负责处理这个 key 的节点，如果该节点不是当前节点
			if addr, peer, notSelf := g.peers.PickPeer(key); notSelf {
				log.Printf("[%v] -> Redirected to key[%v] at %v\n",
					g.peers.Addr(), key, addr)
				// 那么就从远程节点获取缓存
				if value, err := g.getFromPeer(peer, key); err == nil {
					return value, nil
				} else { // 从远程节点获取缓存失败了，可能是因为远程节点已经挂掉了，此时只做日志记录
					log.Printf(
						"[%v]get from peer[%v] error: %v, try to get from local",
						g.peers.Addr(), addr, err)
				}
			}
		}
		// 走到这里说明是以下几种情况：
		// - 没有远程节点（单机环境）
		// - 负责处理该 key 的就是当前节点
		// - 无法从远程节点获取到缓存
		// 这几种情况都需要当前节点从数据源获取数据，并添加到缓存
		return g.getFromLocally(key)
	})
	if err == nil {
		value = v.(*ByteView)
	}
	return
}

// 从远程节点获取数据
func (g *Group) getFromPeer(peer PeerGetter, key string) (*ByteView, error) {
	req := &cachepb.Request{Key: key, Group: g.name}
	resp := &cachepb.Response{}
	if err := peer.Get(req, resp); err != nil {
		return &ByteView{}, err
	}
	return &ByteView{b: resp.Value}, nil
}

// getFromLocally 通过调用 g.getter 从本地获得数据，同时添加到缓存
func (g *Group) getFromLocally(key string) (val *ByteView, err error) {
	log.Printf("[%v] get from locally\n", g.peers.Addr())
	v, err := g.getter.Get(key)
	if err != nil {
		return nil, err
	}
	val = &ByteView{b: v}
	// 获取到同时添加到缓存中
	g.addCache(key, val)
	return
}

// 将 cache 添加到 mainCache 中
func (g *Group) addCache(key string, val *ByteView) {
	g.mainCache.Add(key, val)
}
