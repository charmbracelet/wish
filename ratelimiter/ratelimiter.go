// Package ratelimiter provides basic rate limiting functionality as a with middeware.
package ratelimiter

import (
	"errors"
	"log"
	"sync"

	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
	"golang.org/x/time/rate"
)

// ErrRateLimitExceeded happens when the connection was denied due to the rate limit being exceeded.
var ErrRateLimitExceeded = errors.New("rate limit exceeded, please try again later")

// KeyFn defines how the rate limit key should be composed.
type KeyFn func(s ssh.Session) string

// NewLimiterFn should construct a new rate limiter instance.
type NewLimiterFn func() *rate.Limiter

// DefaultKeyFn is the default rate limiter key implementation.
func DefaultKeyFn(s ssh.Session) string {
	return s.RemoteAddr().String()
}

// Middleware provides a new rate limiting Middleware.
func Middleware(
	limiterFn NewLimiterFn,
	keyFn KeyFn,
) wish.Middleware {
	limiters := newLimiters(limiterFn)
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			if err := limiters.Allow(keyFn(s)); err != nil {
				wish.Fatal(s, err)
				return
			}

			sh(s)
		}
	}
}

type limiters struct {
	mut   sync.Mutex
	rates map[string]*rate.Limiter
	fn    NewLimiterFn
}

func newLimiters(fn NewLimiterFn) *limiters {
	return &limiters{
		rates: map[string]*rate.Limiter{},
		fn:    fn,
	}
}

func (r *limiters) Allow(key string) error {
	r.mut.Lock()
	defer r.mut.Unlock()

	limiter, ok := r.rates[key]
	if !ok {
		limiter = r.fn()
	}
	allowed := limiter.Allow()
	r.rates[key] = limiter
	log.Printf("rate limiter key: %q, allowed? %v", key, allowed)
	if allowed {
		return nil
	}
	log.Println(limiter)
	return ErrRateLimitExceeded
}
