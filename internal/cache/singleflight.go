package cache

import "sync"

type call struct {
	wg  sync.WaitGroup
	val string
	ok  bool
	err error
}

type flightGroup struct {
	mu sync.Mutex
	m  map[string]*call
}

func (g *flightGroup) do(key string, fn func() (string, bool, error)) (string, bool, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.ok, c.err
	}

	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.ok, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.ok, c.err
}
