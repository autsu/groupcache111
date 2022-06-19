package cache

import (
	"cache/pb/cachepb"
)

// PeerPicker 因为是分布式系统，所以缓存可能存储在任意一台机器上（没有实现一致性），
// 通过 key 可以在 hash 环上找到对应的节点，这个节点可能是
type PeerPicker interface {
	// PickPeer 看看当前这个 key 应该交给哪个节点进行处理，返回该节点的 PeerGetter
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 从某个节点中获取缓存
type PeerGetter interface {
	// Get 用于从对应 group 查找缓存值
	Get(in *cachepb.Request, out *cachepb.Response) error
}
