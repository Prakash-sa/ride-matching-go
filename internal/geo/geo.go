package geo

import (
	"math"
	"sync"
	"time"

	"github.com/example/ride-matching/internal/models"
)

// Geo is the minimal interface required by the matcher and handlers.
type Geo interface {
	Nearby(lat, lon float64, limit int) []models.Driver
	Upsert(d models.Driver)
}

type Index struct {
	mu      sync.RWMutex
	drivers map[string]models.Driver
}

func NewIndex() *Index {
	return &Index{drivers: make(map[string]models.Driver)}
}

func (g *Index) Upsert(d models.Driver) {
	g.mu.Lock()
	defer g.mu.Unlock()
	d.Updated = time.Now()
	g.drivers[d.ID] = d
}

// naive scan; in prod use geo-hash or H3
func (g *Index) Nearby(lat, lon float64, limit int) []models.Driver {
	g.mu.RLock()
	defer g.mu.RUnlock()
	type pair struct {
		d    models.Driver
		dist float64
	}
	arr := make([]pair, 0, len(g.drivers))
	for _, d := range g.drivers {
		if !d.Online {
			continue
		}
		dist := Haversine(lat, lon, d.Loc.Lat, d.Loc.Lon)
		arr = append(arr, pair{d, dist})
	}
	// partial selection sort for top-N
	n := limit
	if n > len(arr) {
		n = len(arr)
	}
	for i := 0; i < n; i++ {
		minIdx := i
		for j := i + 1; j < len(arr); j++ {
			if arr[j].dist < arr[minIdx].dist {
				minIdx = j
			}
		}
		arr[i], arr[minIdx] = arr[minIdx], arr[i]
	}
	out := make([]models.Driver, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, arr[i].d)
	}
	return out
}

// Haversine distance in meters
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
