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

// NewRateLimiter returns a new RateLimiter that allows events up to rate r
// and permits bursts of at most b tokens.
// It creates one in-memory map of rate.Limiter for each remote address.
func NewRateLimiter(r rate.Limit, b int) RateLimiter {
	return &limiters{
		rates: map[string]*rate.Limiter{},
		r:     r,
		b:     b,
	}
}

type limiters struct {
	mut   sync.Mutex
	rates map[string]*rate.Limiter
	r     rate.Limit
	b     int
}

func (r *limiters) Allow(s ssh.Session) error {
	r.mut.Lock()
	defer r.mut.Unlock()

	key := s.RemoteAddr().String()

	limiter, ok := r.rates[key]
	if !ok {
		limiter = rate.NewLimiter(r.r, r.b)
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
