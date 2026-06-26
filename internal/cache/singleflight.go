package cache

import (
	"log"
	"sync"
)

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
		log.Printf("[cache/singleflight] Coalescing request for key %q — waiting on active execution", key)
		g.mu.Unlock()
		c.wg.Wait()
		log.Printf("[cache/singleflight] Request for key %q completed using shared results from active execution", key)
		return c.val, c.ok, c.err
	}

	log.Printf("[cache/singleflight] Registering active execution for key %q", key)
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.ok, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	log.Printf("[cache/singleflight] Finished execution for key %q, cleaned up registration", key)
	return c.val, c.ok, c.err
}
