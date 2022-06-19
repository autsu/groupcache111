package lru

import (
	"container/list"
	"fmt"
	"testing"
)

func printList(l *list.List) {
	for i := l.Front(); i != nil; i = i.Next() {
		v := i.Value.(*entry)
		fmt.Printf("%v,%v -> ", v.key, v.value)
	}
	fmt.Println()
}

type Str struct {
	s string
}

func (s *Str) Len() int64 {
	return int64(len(s.s))
}

func TestLru(t *testing.T) {
	lru := New(5, nil)

	lru.Add("1", &Str{s: "111"})
	fmt.Printf("[add {1, 111}]  curBytes: %v, maxBytes: %v, len: %v \n",
		lru.curBytes, lru.maxBytes, lru.Len())
	printList(lru.ll)
	fmt.Println("========================================")
	//fmt.Println(LRU.Get(1))

	lru.Add("2", &Str{s: "222"})
	fmt.Printf("[add {2, 222}]  curBytes: %v, maxBytes: %v, len: %v \n",
		lru.curBytes, lru.maxBytes, lru.Len())
	printList(lru.ll)
	fmt.Println("========================================")

	fmt.Println(lru.Get("1"))
	printList(lru.ll)
	fmt.Println("========================================")

	lru.Add("3", &Str{s: "333"})
	fmt.Printf("[add {3, 333}]  curBytes: %v, maxBytes: %v, len: %v \n",
		lru.curBytes, lru.maxBytes, lru.Len())
	printList(lru.ll)
	fmt.Println("========================================")

	fmt.Println(lru.Get("2"))
	printList(lru.ll)
	fmt.Println("========================================")

	lru.Add("4", &Str{s: "444"})
	fmt.Printf("[add {4, 444}]  curBytes: %v, maxBytes: %v, len: %v \n",
		lru.curBytes, lru.maxBytes, lru.Len())
	fmt.Println(lru.Get("1"))
	fmt.Println("========================================")

	fmt.Println(lru.Get("3"))
	fmt.Println("========================================")

	fmt.Println(lru.Get("4"))
	fmt.Println("========================================")
}
