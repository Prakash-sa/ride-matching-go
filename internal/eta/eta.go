package eta

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/example/ride-matching/internal/models"
)

// Client is the interface used by the matcher to get ETAs.
type Client interface {
	EstimateSeconds(from, to models.Coord) (float64, error)
}

// Cache is a tiny in-memory cache for ETA lookups keyed by coords.
type Cache struct {
	mu    sync.RWMutex
	store map[string]cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	v  float64
	ts time.Time
}

// NewCache creates a cache with the provided TTL.
func NewCache(ttl time.Duration) *Cache {
	return &Cache{store: make(map[string]cacheEntry), ttl: ttl}
}

func keyFor(a, b models.Coord) string {
	return fmtCoord(a) + "->" + fmtCoord(b)
}

func fmtCoord(c models.Coord) string {
	return fmt.Sprintf("%.6f,%.6f", c.Lat, c.Lon)
}

// Get returns cached value and true if present and not expired.
func (c *Cache) Get(a, b models.Coord) (float64, bool) {
	k := keyFor(a, b)
	c.mu.RLock()
	e, ok := c.store[k]
	c.mu.RUnlock()
	if !ok {
		return 0, false
	}
	if time.Since(e.ts) > c.ttl {
		c.mu.Lock()
		delete(c.store, k)
		c.mu.Unlock()
		return 0, false
	}
	return e.v, true
}

// Set stores a value in the cache.
func (c *Cache) Set(a, b models.Coord, v float64) {
	k := keyFor(a, b)
	c.mu.Lock()
	c.store[k] = cacheEntry{v: v, ts: time.Now()}
	c.mu.Unlock()
}

// Naive ETA: distance / speed_mps. In prod use a routing engine.
func EstimateSeconds(from, to models.Coord, speedMps float64) float64 {
	if speedMps <= 0 {
		speedMps = 8.0 // ~28.8 km/h default city speed
	}
	// reuse geo.Haversine by local import (copy) to avoid cycles. Recompute here simply.
	d := haversine(from.Lat, from.Lon, to.Lat, to.Lon)
	return d / speedMps
}

// local haversine to avoid import cycle
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000.0
	toRad := func(deg float64) float64 { return deg * math.Pi / 180.0 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
