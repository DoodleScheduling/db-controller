package vault

type Cache struct {
	cache map[string]*Vault
}

func NewCache() *Cache {
	c := make(map[string]*Vault)
	return &Cache{
		cache: c,
	}
}

func (c *Cache) Get(host string) (*Vault, error) {
	if _, ok := c.cache[host]; !ok {
		if cache, err := NewVault(host); err != nil {
			return nil, err
		} else {
			c.cache[host] = cache
		}
	}
	return c.cache[host], nil
}

func (c *Cache) Remove(host string) {
	delete(c.cache, host)
}
