package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"hash/fnv"
	"net/http"
	"net/netip"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ContextKeyIPHash is where the anonymous per-request IP hash is stored.
// Downstream handlers read it with IPHash(c) instead of touching the raw IP.
const ContextKeyIPHash = "ip_hash"

// shardCount must be a power of two. 32 is plenty for a few thousand rps.
const shardCount = 32

// RateLimitConfig configures the limiter. Zero values fall back to defaults.
type RateLimitConfig struct {
	// Secret keys the HMAC used to hash IPs. Required, at least 32 random bytes.
	// A plain SHA-256 of an IP is reversible by brute force; HMAC with a secret
	// is not.
	Secret []byte

	// Rate is the sustained request rate allowed per IP hash.
	Rate rate.Limit

	// Burst is how many requests may arrive at once before Rate applies.
	Burst int

	// IdleTTL is how long an unused entry is kept before the sweeper drops it.
	IdleTTL time.Duration

	// SweepEvery is how often the sweeper runs.
	SweepEvery time.Duration

	// MaxEntriesPerShard bounds memory. When a shard is full and pruning frees
	// nothing, further new IPs are rejected rather than allowed, so a flood from
	// many addresses cannot grow the map without limit. 0 disables the cap.
	MaxEntriesPerShard int

	// SkipPaths are route patterns (as returned by c.FullPath()) that bypass the
	// limiter entirely, e.g. "/healthz".
	SkipPaths []string

	// IPv6PrefixBits is how much of an IPv6 address is kept before hashing.
	// A single residential IPv6 customer is handed a whole /64 (or wider), so
	// limiting per full address lets one visitor rotate through trillions of
	// them for free. 64 is the right default; drop to 56 or 48 to also cover
	// customers delegated a larger block, at the cost of grouping more
	// unrelated users together. Ignored for IPv4.
	IPv6PrefixBits int
}

func (c *RateLimitConfig) applyDefaults() {
	if c.Rate == 0 {
		c.Rate = rate.Every(time.Second) // 1 req/sec sustained
	}
	if c.Burst == 0 {
		c.Burst = 20
	}
	if c.IdleTTL == 0 {
		c.IdleTTL = 10 * time.Minute
	}
	if c.SweepEvery == 0 {
		c.SweepEvery = 2 * time.Minute
	}
	if c.MaxEntriesPerShard == 0 {
		c.MaxEntriesPerShard = 50_000 // ~1.6M IPs across 32 shards
	}
	if c.IPv6PrefixBits <= 0 || c.IPv6PrefixBits > 128 {
		c.IPv6PrefixBits = 64
	}
}

type entry struct {
	lim  *rate.Limiter
	seen time.Time
}

type shard struct {
	mu      sync.Mutex
	entries map[string]*entry
}

// Limiter is a token-bucket rate limiter keyed by a rotating hash of the
// client IP. It is safe for concurrent use.
type Limiter struct {
	cfg    RateLimitConfig
	shards [shardCount]*shard
	skip   map[string]struct{}

	keyMu  sync.RWMutex
	keyDay string
	key    []byte

	stop     chan struct{}
	stopOnce sync.Once
}

// NewLimiter builds a Limiter and starts its background sweeper.
// Call Close during shutdown to stop the sweeper goroutine.
func NewLimiter(cfg RateLimitConfig) *Limiter {
	cfg.applyDefaults()

	l := &Limiter{
		cfg:  cfg,
		skip: make(map[string]struct{}, len(cfg.SkipPaths)),
		stop: make(chan struct{}),
	}
	for i := range l.shards {
		l.shards[i] = &shard{entries: make(map[string]*entry)}
	}
	for _, p := range cfg.SkipPaths {
		l.skip[p] = struct{}{}
	}

	go l.sweepLoop()
	return l
}

// Close stops the sweeper. Safe to call more than once.
func (l *Limiter) Close() {
	l.stopOnce.Do(func() { close(l.stop) })
}

// dailyKey derives a per-day subkey from the master secret. Hashes therefore
// cannot be correlated across days, which keeps the limiter's state
// pseudonymous while rate limiting still works within a day.
func (l *Limiter) dailyKey() []byte {
	day := time.Now().UTC().Format("2006-01-02")

	l.keyMu.RLock()
	if l.keyDay == day {
		k := l.key
		l.keyMu.RUnlock()
		return k
	}
	l.keyMu.RUnlock()

	l.keyMu.Lock()
	defer l.keyMu.Unlock()
	if l.keyDay != day {
		m := hmac.New(sha256.New, l.cfg.Secret)
		m.Write([]byte(day))
		l.key = m.Sum(nil)
		l.keyDay = day
	}
	return l.key
}

// normalizeAddr reduces an address to the unit that gets one budget: the exact
// address for IPv4, the routing prefix for IPv6. Unparseable input is passed
// through unchanged so it still gets limited rather than skipped.
func (l *Limiter) normalizeAddr(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}
	addr = addr.Unmap().WithZone("") // ::ffff:1.2.3.4 -> 1.2.3.4; drop %eth0

	if addr.Is4() {
		return addr.String()
	}

	prefix, err := addr.Prefix(l.cfg.IPv6PrefixBits)
	if err != nil {
		return addr.String()
	}
	return prefix.String()
}

// HashIP returns the anonymous identifier for an IP. The digest is truncated to
// 16 bytes, which is far beyond collision risk here and halves key size.
func (l *Limiter) HashIP(ip string) string {
	m := hmac.New(sha256.New, l.dailyKey())
	m.Write([]byte(l.normalizeAddr(ip)))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil)[:16])
}

func (l *Limiter) shardFor(hash string) *shard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(hash))
	return l.shards[h.Sum32()&(shardCount-1)]
}

// Allow reports whether a request from this hash may proceed. When it may not,
// the returned duration is how long the caller should wait before retrying.
func (l *Limiter) Allow(hash string) (bool, time.Duration) {
	now := time.Now()
	s := l.shardFor(hash)

	s.mu.Lock()
	e, ok := s.entries[hash]
	if !ok {
		if l.cfg.MaxEntriesPerShard > 0 && len(s.entries) >= l.cfg.MaxEntriesPerShard {
			s.pruneLocked(now.Add(-l.cfg.IdleTTL))
			if len(s.entries) >= l.cfg.MaxEntriesPerShard {
				s.mu.Unlock()
				return false, time.Minute // shard full: shed load rather than grow
			}
		}
		e = &entry{lim: rate.NewLimiter(l.cfg.Rate, l.cfg.Burst)}
		s.entries[hash] = e
	}
	e.seen = now
	s.mu.Unlock()

	// Reserve rather than Allow so a rejected caller learns how long to wait.
	res := e.lim.ReserveN(now, 1)
	if !res.OK() {
		return false, time.Second
	}
	if d := res.DelayFrom(now); d > 0 {
		res.CancelAt(now)
		return false, d
	}
	return true, 0
}

// Middleware hashes the client IP, stores it on the context and enforces the
// per-IP budget.
func (l *Limiter) IpRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, skip := l.skip[c.FullPath()]; skip {
			c.Next()
			return
		}

		ip := c.ClientIP()
		if ip == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "cannot determine client address"})
			return
		}

		hash := l.HashIP(ip)
		c.Set(ContextKeyIPHash, hash)

		allowed, retry := l.Allow(hash)
		if !allowed {
			secs := int(retry/time.Second) + 1
			c.Header("Retry-After", strconv.Itoa(secs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Set("iphash", hash)
		c.Next()
	}
}

// IPHash returns the anonymous IP hash set by the middleware. Use it to key
// session tokens or per-source counters; never store it with an event.
func IPHash(c *gin.Context) (string, bool) {
	v, ok := c.Get(ContextKeyIPHash)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// Len reports how many IPs are currently tracked. Useful as a gauge metric.
func (l *Limiter) Len() int {
	n := 0
	for _, s := range l.shards {
		s.mu.Lock()
		n += len(s.entries)
		s.mu.Unlock()
	}
	return n
}

func (l *Limiter) sweepLoop() {
	t := time.NewTicker(l.cfg.SweepEvery)
	defer t.Stop()

	for {
		select {
		case <-l.stop:
			return
		case now := <-t.C:
			cutoff := now.Add(-l.cfg.IdleTTL)
			for _, s := range l.shards {
				s.mu.Lock()
				s.pruneLocked(cutoff)
				s.mu.Unlock()
			}
		}
	}
}

// pruneLocked drops entries untouched since cutoff. Caller holds s.mu.
func (s *shard) pruneLocked(cutoff time.Time) {
	for k, e := range s.entries {
		if e.seen.Before(cutoff) {
			delete(s.entries, k)
		}
	}
}
