package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"taskapi/internal/response"
)

const (
	clientTTL       = 3 * time.Minute // drop the limiter of an IP not seen for this long
	cleanupInterval = time.Minute
)

// RateLimiter is a token bucket per client IP: it refills rps tokens per second, up to burst tokens.
// State is in memory and applies to a single instance; multiple instances would need shared storage such as Redis.
type RateLimiter struct {
	rps   rate.Limit
	burst int

	mu      sync.Mutex
	clients map[string]*client

	stop     chan struct{}
	stopOnce sync.Once
}

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates the limiter and starts a background goroutine that removes idle IPs, so the map does not grow forever.
// Call Close to stop the goroutine.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		rps:     rate.Limit(rps),
		burst:   burst,
		clients: make(map[string]*client),
		stop:    make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

// RateLimit limits next with rl: over the limit it returns 429 with a Retry-After header saying when to retry.
// The argument order follows the standard library's http.MaxBytesHandler(h, n): the wrapped handler first, then the settings.
func RateLimit(next http.Handler, rl *RateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := rl.limiterFor(clientIP(r)).Reserve()
		if delay := res.Delay(); delay > 0 {
			res.Cancel() // not serving this request, so give the reserved token back
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(delay.Seconds()))))
			response.WriteError(w, r, response.ErrRateLimited)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Close stops the background cleanup goroutine. It is safe to call more than once.
func (rl *RateLimiter) Close() {
	rl.stopOnce.Do(func() { close(rl.stop) })
}

func (rl *RateLimiter) limiterFor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	c, ok := rl.clients[ip]
	if !ok {
		c = &client{limiter: rate.NewLimiter(rl.rps, rl.burst)}
		rl.clients[ip] = c
	}
	c.lastSeen = time.Now()
	return c.limiter
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stop:
			return
		case now := <-ticker.C:
			rl.mu.Lock()
			for ip, c := range rl.clients {
				if now.Sub(c.lastSeen) > clientTTL {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// clientIP returns the source IP of the connection.
// X-Forwarded-For is ignored on purpose: clients can forge it to bypass the limit.
// Behind a trusted proxy or load balancer, switch to the header that proxy sets.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
