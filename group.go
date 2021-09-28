package cache

import (
	"fmt"
	"log"
	"sync"
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

type Size int64

const (
	B  Size = iota + 1
	KB      = 1 << (10 * iota)
	MB
	GB
)

var (
	groups = make(map[string]*Group)
	mu     sync.RWMutex
)

// Group 是一个缓存的命名空间
type Group struct {
	name      string // 全局唯一
	getter    Getter // 缓存未命中时获取源数据的回调
	mainCache cache
	peers     PeerPicker
}

func NewGroup(name string, size Size, getter Getter) *Group {
	if getter == nil {
		panic("getter cannot be nil")
	}

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{size: int64(size)},
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
		log.Println("cache is hit")
		return val, nil
	}
	// 缓存中不存在，则去指定的数据源中获取
	return g.load(key)
}

// load 当缓存不在当前节点时调用该方法
func (g *Group) load(key string) (value *ByteView, err error) {
	// 如果有远程节点，则需要确定这个 key 应该交给哪个节点进行处理（负载均衡）
	if g.peers != nil {
		// 确定负责处理这个 key 的节点，如果该节点不是当前节点
		if peer, ok := g.peers.PickPeer(key); ok {
			// 那么就从远程节点获取缓存
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
			log.Println("failed to get from peer", err)
		}
	}
	// 走到这里说明没有远程节点（单机环境），或者负责处理该 key 的就是当前节点，那么由
	// 当前节点负责，从指定数据源获取数据，并添加到缓存
	return g.getFromLocally(key)
}

// 从远程节点获取数据
func (g *Group) getFromPeer(peer PeerGetter, key string) (*ByteView, error) {
	//
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return &ByteView{}, err
	}
	return &ByteView{b: bytes}, nil
}

// getFromLocally 通过调用 g.getter 从本地获得数据，同时添加到缓存
func (g *Group) getFromLocally(key string) (val *ByteView, err error) {
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
