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

func (c *Cache) Get(host string, rootUsername string, rootPassword string, rootAuthenticationDatabase string) (*PostgreSQLServer, error) {
	if _, ok := c.cache[host]; !ok {
		if server, err := NewPostgreSQLServer(host, rootUsername, rootPassword, rootAuthenticationDatabase); err != nil {
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
		server.Close()
	}
	delete(c.cache, host)
}
