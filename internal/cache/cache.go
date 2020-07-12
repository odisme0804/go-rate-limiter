package cache

//go:generate mockgen -destination=./mock/mockCache.go -package=mockCache github.com/odisme0804/go-rate-limiter/cache Cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/patrickmn/go-cache"
)

type Cache interface {
	Get(ctx context.Context, key string, ptrValue interface{}) error
	Set(ctx context.Context, key string, value interface{}, expires time.Duration) error
}

type InMemoryCache struct {
	cache *cache.Cache
}

func NewInMemoryCache(defaultExpiration time.Duration) Cache {
	return &InMemoryCache{
		cache: cache.New(defaultExpiration, time.Minute),
	}
}

func (c *InMemoryCache) Get(ctx context.Context, key string, ptrValue interface{}) error {
	value, found := c.cache.Get(key)
	if !found {
		return ErrCacheMiss
	}
	bytes, ok := value.([]byte)
	if !ok {
		return ErrCacheData
	}

	return json.Unmarshal(bytes, ptrValue)
}

func (c *InMemoryCache) Set(ctx context.Context, key string, value interface{}, expires time.Duration) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	c.cache.Set(key, bytes, expires)
	return nil
}
