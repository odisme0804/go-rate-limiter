package limiter

import "errors"

var (
	ErrInternal   = errors.New("rateLimiter: internal")
	ErrReachLimit = errors.New("rateLimiter: reach request limit")
)
