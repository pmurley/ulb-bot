package cache

import (
	"sync"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/pmurley/ulb-bot/internal/models"
)

type Cache struct {
	cache    *gocache.Cache
	mu       sync.RWMutex
	duration time.Duration
}

func New(duration time.Duration) *Cache {
	return &Cache{
		cache:    gocache.New(duration, duration*2),
		duration: duration,
	}
}

func (c *Cache) SetPlayers(players []models.Player) {
	c.cache.Set("players", players, c.duration)
}

func (c *Cache) GetPlayers() (models.PlayerList, bool) {
	if players, found := c.cache.Get("players"); found {
		return models.PlayerList(players.([]models.Player)), true
	}
	return nil, false
}

func (c *Cache) Flush() {
	c.cache.Flush()
}