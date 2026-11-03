package transport

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter ограничивает число запросов от одного IP-адреса за интервал времени.
type RateLimiter struct {
	maximumRequests int
	window          time.Duration

	mu          sync.Mutex
	clients     map[string]clientWindow
	lastCleanup time.Time
}

type clientWindow struct {
	startedAt time.Time
	requests  int
}

// NewRateLimiter создаёт ограничитель с фиксированным окном запросов.
func NewRateLimiter(maximumRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		maximumRequests: maximumRequests,
		window:          window,
		clients:         make(map[string]clientWindow),
	}
}

func (l *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, retryAfter := l.allow(clientIP(r), time.Now())
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			writeError(w, http.StatusTooManyRequests, "Превышен лимит запросов", "Повторите запрос позднее")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (l *RateLimiter) allow(client string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.lastCleanup.IsZero() || now.Sub(l.lastCleanup) >= l.window {
		for key, window := range l.clients {
			if now.Sub(window.startedAt) >= l.window {
				delete(l.clients, key)
			}
		}
		l.lastCleanup = now
	}

	window := l.clients[client]
	if window.startedAt.IsZero() || now.Sub(window.startedAt) >= l.window {
		window = clientWindow{startedAt: now}
	}
	if window.requests >= l.maximumRequests {
		return false, l.window - now.Sub(window.startedAt)
	}

	window.requests++
	l.clients[client] = window
	return true, 0
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}
