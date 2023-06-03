package groupcache

import (
	"cache/pb/cachepb"
)

// PeerPicker 是一个节点选择器
// 因为是分布式系统，所以缓存可能存储在任意一台机器上（没有实现一致性），
// 通过 key 加上一致性哈希算法可以定位到一个节点
type PeerPicker interface {
	Addr() string
	// PickPeer 看看当前这个 key 应该交给哪个节点进行处理，返回该节点的 PeerGetter
	PickPeer(key string) (addr string, peer PeerGetter, notSelf bool)
}

// PeerGetter 从某个节点中获取缓存
type PeerGetter interface {
	// Get 用于从对应 group 查找缓存值
	Get(in *cachepb.Request, out *cachepb.Response) error
}
