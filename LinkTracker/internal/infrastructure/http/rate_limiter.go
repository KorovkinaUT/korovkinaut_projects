package httpinfra

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/config"
)

type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

func NewRateLimiter(cfg *config.RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(cfg.RPS),
		burst:    cfg.Burst,
	}
}

func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		limiter := l.getLimiter(ip)
		if !limiter.Allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// If limiter already exists, returns it, else creates new limiter
func (l *RateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, ok := l.limiters[ip]
	if !ok {
		limiter = rate.NewLimiter(l.rps, l.burst)
		l.limiters[ip] = limiter
	}

	return limiter
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
