package cache

import (
	"sync"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/pmurley/ulb-bot/internal/models"
)

type Cache struct {
	cache        *gocache.Cache
	mu           sync.RWMutex
	isLoading    bool
	lastLoadTime time.Time
}

func New(duration time.Duration) *Cache {
	return &Cache{
		cache: gocache.New(gocache.NoExpiration, 5*time.Minute),
	}
}

func (c *Cache) SetPlayers(players []models.Player) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Set("players", players, gocache.NoExpiration)
	c.lastLoadTime = time.Now()
	c.isLoading = false
}

func (c *Cache) GetPlayers() (models.PlayerList, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if players, found := c.cache.Get("players"); found {
		return models.PlayerList(players.([]models.Player)), true
	}
	return nil, false
}

func (c *Cache) SetLoading(loading bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isLoading = loading
}

func (c *Cache) IsLoading() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isLoading
}

func (c *Cache) GetLastLoadTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastLoadTime
}

func (c *Cache) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Flush()
}
