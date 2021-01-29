package postgresql

type Cache struct {
	cache map[string]*PostgreSQLServer
}

func NewCache() *Cache {
	c := make(map[string]*PostgreSQLServer)
	return &Cache{
		cache: c,
	}
}

func (c *Cache) Get(host string, port string, rootUsername string, rootPassword string) (*PostgreSQLServer, error) {
	if _, ok := c.cache[host]; !ok {
		if server, err := NewPostgreSQLServer(host, port, rootUsername, rootPassword); err != nil {
			return nil, err
		} else {
			c.cache[host] = server
		}
	}
	return c.cache[host], nil
}

func (c *Cache) Remove(host string) {
	delete(c.cache, host)
}
