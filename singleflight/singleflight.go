package singleflight

import "sync"

type call struct {
	value any
	err   error
	wg    sync.WaitGroup
}

type Group struct {
	sync.Mutex
	m map[string]*call
}

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
