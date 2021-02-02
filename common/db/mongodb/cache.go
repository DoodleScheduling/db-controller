package mongodb

type Cache struct {
	cache map[string]*MongoDBServer
}

func NewCache() *Cache {
	c := make(map[string]*MongoDBServer)
	return &Cache{
		cache: c,
	}
}

func (c *Cache) Get(host string, rootUsername string, rootPassword string, rootAuthenticationDatabase string) (*MongoDBServer, error) {
	if _, ok := c.cache[host]; !ok {
		if server, err := NewMongoDBServer(host, rootUsername, rootPassword, rootAuthenticationDatabase); err != nil {
			return nil, err
		} else {
			c.cache[host] = server
		}
	}
	return c.cache[host], nil
}

func (c *Cache) Remove(host string) {
	server := c.cache[host]
	if server != nil {
		_ = server.Close()
	}
	delete(c.cache, host)
}
