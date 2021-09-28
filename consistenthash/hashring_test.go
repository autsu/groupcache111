package consistenthash

import (
	"fmt"
	"strconv"
	"testing"
)

func TestHash(t *testing.T) {
	// 每个节点有 3 个虚拟节点
	m := New(3, func(data []byte) uint32 {
		v, _ := strconv.Atoi(string(data))
		return uint32(v)
	})

	// 添加三个真实节点
	m.Add("6", "4", "2")
	fmt.Printf("添加后，hash 环上的节点：\n %v \n", m.nodes)
	fmt.Println("对应的映射关系: ")
	for virtualNode, realNode := range m.hashMap {
		fmt.Printf("【虚拟节点】%v 对应【真实节点】%v \n", virtualNode, realNode)
	}


	testCase := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	for k, v := range testCase {
		if m.Get(k) != v {
			t.Errorf("Asking for %v, should have yielded %s", k, v)
		}
	}


}
