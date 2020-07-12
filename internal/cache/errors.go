package cache

import "errors"

var (
	ErrCacheMiss = errors.New("cache: miss")
	ErrCacheData = errors.New("cache: stored data is not supported")
)
