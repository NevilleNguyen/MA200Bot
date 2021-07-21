package app

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var (
	DefaultRateLimitTimeout = 2 * time.Second
)

// RateLimiter do rate limiter
type RateLimiter struct {
	sync.RWMutex
	limiter *rate.Limiter
}

// rate is number of request per seconds
func NewRateLimiter(rateLimit int, burst int) *RateLimiter {
	limiter := rate.NewLimiter(rate.Limit(rateLimit), burst)

	return &RateLimiter{
		limiter: limiter,
	}
}

// WaitN waits until enough resources are available for a request with given weight.
func (r *RateLimiter) WaitN(timeout time.Duration, weight int) error {
	r.Lock()
	defer r.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := r.limiter.WaitN(ctx, weight); err != nil {
		return err
	}
	return nil
}
