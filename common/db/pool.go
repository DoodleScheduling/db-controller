package db

import (
	"context"
	"fmt"
	"sync"
)

type ClientPool struct {
	pool  map[string]Handler
	mutex sync.Mutex
}

func NewClientPool() *ClientPool {
	return &ClientPool{
		pool: make(map[string]Handler),
	}
}

func (c *ClientPool) FromURI(ctx context.Context, invoke Invoke, uri, database, username, password string) (Handler, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	key := fmt.Sprintf("%s/%s/%s/%s", uri, database, username, password)

	if _, ok := c.pool[key]; !ok {
		if server, err := invoke(ctx, uri, database, username, password); err != nil {
			return nil, err
		} else {
			c.pool[key] = server
		}
	}

	return c.pool[key], nil
}
