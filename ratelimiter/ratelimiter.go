// Package ratelimiter provides basic rate limiting functionality as a with middleware.
//
// It limits the amount of connections a source can make in a specified amount of time.
package ratelimiter

import (
	"errors"
	"net"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/time/rate"
)

// ErrRateLimitExceeded happens when the connection was denied due to the rate limit being exceeded.
var ErrRateLimitExceeded = errors.New("rate limit exceeded, please try again later")

// RateLimiter implementations should check if a given session is allowed to
// proceed or not, returning an error if they aren't.
// Its up to the implementation to handle what identifies an session as well
// as the implementation details of these limits.
type RateLimiter interface {
	Allow(s ssh.Session) error
}

// Middleware provides a new rate limiting Middleware.
func Middleware(limiter RateLimiter) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			if err := limiter.Allow(s); err != nil {
				wish.Fatal(s, err)
				return
			}

			sh(s)
		}
	}
}

// NewRateLimiter returns a new RateLimiter that allows events up to rate rate,
// permits bursts of at most burst tokens and keeps a cache of maxEntries
// limiters.
//
// Internally, it creates a LRU Cache of *rate.Limiter, in which the key is
// the remote IP address.
func NewRateLimiter(r rate.Limit, burst int, maxEntries int) RateLimiter {
	if maxEntries <= 0 {
		maxEntries = 1
	}
	// only possible error is if maxEntries is <= 0, which is prevented above.
	cache, _ := lru.New[string, *rate.Limiter](maxEntries)
	return &limiters{
		rate:  r,
		burst: burst,
		cache: cache,
	}
}

type limiters struct {
	cache *lru.Cache[string, *rate.Limiter]
	rate  rate.Limit
	burst int
}

func (r *limiters) Allow(s ssh.Session) error {
	var key string
	switch addr := s.RemoteAddr().(type) {
	case *net.TCPAddr:
		key = addr.IP.String()
	default:
		key = addr.String()
	}

	var allowed bool
	limiter, ok := r.cache.Get(key)
	if ok {
		allowed = limiter.Allow()
	} else {
		limiter := rate.NewLimiter(r.rate, r.burst)
		allowed = limiter.Allow()
		r.cache.Add(key, limiter)
	}

	log.Debug("rate limiter key", "key", key, "allowed", allowed)
	if allowed {
		return nil
	}
	return ErrRateLimitExceeded
}
