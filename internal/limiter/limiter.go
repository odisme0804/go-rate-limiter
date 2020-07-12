package limiter

//go:generate mockgen -destination=./mock/mockLimiter.go -package=mockLimiter go-rate-limiter/internal/limiter Limiter

import (
	"context"
	"errors"
	"sync"
	"time"

	"go-rate-limiter/internal/cache"
)

type Limiter interface {
	Check(key string) (int, int64, error)
	Take(key string) (int, int64, error)
	GetRequestWindow() time.Duration
	GetReuqestLimit() int
}

type TokenBucket struct {
	Count      int
	Expiration int64
}

type TokenBucketLimiter struct {
	sync.RWMutex
	requestWindow time.Duration
	requestLimit  int
	cache         cache.Cache
}

func NewTokenBucketLimiter(window time.Duration, limit int, cache cache.Cache) Limiter {
	return &TokenBucketLimiter{
		requestWindow: window,
		requestLimit:  limit,
		cache:         cache,
	}
}

func (r *TokenBucketLimiter) Check(key string) (int, int64, error) {
	r.RLock()
	defer r.RUnlock()
	bucket := TokenBucket{}
	if err := r.cache.Get(context.TODO(), key, &bucket); err != nil {
		if errors.Is(err, cache.ErrCacheMiss) {
			return 0, time.Now().Add(r.requestWindow).Unix(), nil
		}

		return 0, 0, ErrInternal
	}

	return bucket.Count, bucket.Expiration, nil
}

func (r *TokenBucketLimiter) Take(key string) (int, int64, error) {
	r.Lock()
	defer r.Unlock()
	bucket := TokenBucket{}
	if err := r.cache.Get(context.TODO(), key, &bucket); err != nil {
		if errors.Is(err, cache.ErrCacheMiss) {
			newBucket := TokenBucket{
				Count:      1,
				Expiration: time.Now().Add(r.requestWindow).Unix(),
			}
			if err := r.cache.Set(context.TODO(), key, newBucket, r.requestWindow); err != nil {
				return 0, 0, ErrInternal
			}

			return newBucket.Count, newBucket.Expiration, nil
		}

		return 0, 0, ErrInternal
	}

	if bucket.Count >= r.requestLimit {
		return bucket.Count, bucket.Expiration, ErrReachLimit
	}

	bucket.Count++
	remainDuration := bucket.Expiration - time.Now().Unix()
	if err := r.cache.Set(context.TODO(), key, bucket, time.Second*time.Duration(remainDuration)); err != nil {
		return 0, 0, ErrInternal
	}

	return bucket.Count, bucket.Expiration, nil
}

func (r *TokenBucketLimiter) GetRequestWindow() time.Duration {
	return r.requestWindow
}
func (r *TokenBucketLimiter) GetReuqestLimit() int {
	return r.requestLimit
}
