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
	clientTTL       = 3 * time.Minute // 超過這段時間沒出現的 IP，就把它的 limiter 清掉
	cleanupInterval = time.Minute
)

// RateLimiter 以 client IP 為單位做 token bucket 限流：每秒補 rps 個 token，最多累積 burst 個。
// 狀態存在記憶體，只對單一 instance 有效；部署多份時要改用 Redis 等共用儲存。
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

// NewRateLimiter 建立限流器，並啟動一個背景 goroutine 定期清掉閒置的 IP，避免 map 無限長大。
// 用完要呼叫 Close 停掉背景 goroutine。
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

// RateLimit 用 rl 的額度限制 next：超過限制時回 429，並在 Retry-After header 告訴 client 幾秒後可以再試。
// 參數順序比照標準庫的 http.MaxBytesHandler(h, n)：先放被包的 handler，再放設定。
func RateLimit(next http.Handler, rl *RateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := rl.limiterFor(clientIP(r)).Reserve()
		if delay := res.Delay(); delay > 0 {
			res.Cancel() // 這次請求不處理，把預約的 token 還回去
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(delay.Seconds()))))
			response.WriteError(w, r, response.ErrRateLimited)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Close 停止背景清理的 goroutine，可以重複呼叫。
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

// clientIP 取連線的來源 IP。
// 刻意不看 X-Forwarded-For：client 可以自己偽造這個 header 來繞過限流。
// 若之後放在可信任的 proxy / load balancer 後面，才改成讀 proxy 設定的 header。
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
