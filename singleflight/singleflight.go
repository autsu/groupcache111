package singleflight

import "sync"

type call struct {
	value any
	err   error
	// 这里的 wg 虽然是值类型，但是 call 在 Group.m 中是指针类型，所以不会发生拷贝
	wg sync.WaitGroup
}

type Group struct {
	sync.Mutex
	m map[string]*call
}

// Do 会调用 fn 来获取值，并且确保多个 goroutine 并发调用 Do 时，同一个 key 下只有一个 goroutine 执行 fn，其他 goroutine 会阻塞等待结果，不会调用 fn
// 对应到缓存，fn 是缓存未命中时，从数据源查询值的操作，Do 可以确保相同 key 下的多个并发请求中，只有一个请求会去查询数据源，其他请求会阻塞等待该请求完成，从而
// 避免缓存穿透
func (g *Group) Do(key string, fn func() (any, error)) (any, error) {
	g.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.Unlock()
		c.wg.Wait()
		return c.value, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.Unlock()

	c.value, c.err = fn()
	c.wg.Done()

	g.Lock()
	delete(g.m, key)
	g.Unlock()

	return c.value, c.err
}
