package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// HashFunc 是一个 hash 函数
type HashFunc func(data []byte) uint32

// Map 是一个哈希环
type Map struct {
	hash     HashFunc          // 自定义 hash 函数
	replicas int64             // 虚拟节点倍数，即每个真实节点有几个虚拟节点
	hashMap  map[uint32]string // 虚拟节点与真实节点的映射表，key 是虚拟节点的 hash 值，value 是真实节点的名称
	nodes    []uint32          // hash 环，保存所有节点的 hash 值
}

func New(replicas int64, fn HashFunc) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[uint32]string),
	}
	// 如果没有传入 hash 函数，则默认使用 crc32
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 添加节点到 hash 环
func (h *Map) Add(node ...string) {
	for _, n := range node {
		// 为每个节点创建 replicas 个虚拟节点
		for i := 0; int64(i) < h.replicas; i++ {
			// 计算 hash
			hash := h.hash([]byte(strconv.Itoa(i) + n))
			// 添加到 hash 环
			h.nodes = append(h.nodes, hash)
			// 添加虚拟节点和真实节点的映射关系
			h.hashMap[hash] = n
		}
	}
	// 对环上的哈希值进行排序
	sort.Slice(h.nodes, func(i, j int) bool {
		return h.nodes[i] < h.nodes[j]
	})
}

// Get 会根据 key 来选择一个节点
func (h *Map) Get(key string) string {
	if len(h.nodes) == 0 {
		return ""
	}
	// 计算 hash
	hash := h.hash([]byte(key))

	// 使用二分查找，找到合适的位置
	// sort.Search() : 第一个参数 n 用来指定查找的范围区间，即 [0, n)
	// 返回找到的第一个数所在的下标，如果没找到则返回 n
	index := sort.Search(len(h.nodes), func(i int) bool {
		// 查找第一个大于等于 hash(key) 的节点
		return h.nodes[i] >= hash
	})

	// 没找到，则返回第一个节点
	if index == len(h.nodes) {
		index = 0
	}
	return h.hashMap[h.nodes[index]]
}
