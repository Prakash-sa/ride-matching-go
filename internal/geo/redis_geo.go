package geo

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/example/ride-matching/internal/models"
	"github.com/redis/go-redis/v9"
)

// RedisGeo implements Geo using Redis GEO commands.
type RedisGeo struct {
	client *redis.Client
	key    string
	ctx    context.Context
}

func NewRedisGeo(addr, password, key string) *RedisGeo {
	c := redis.NewClient(&redis.Options{Addr: addr, Password: password})
	return &RedisGeo{client: c, key: key, ctx: context.Background()}
}

func (r *RedisGeo) Upsert(d models.Driver) {
	// store as GEOADD and HMSET for metadata
	_, _ = r.client.GeoAdd(r.ctx, r.key, &redis.GeoLocation{Longitude: d.Loc.Lon, Latitude: d.Loc.Lat, Name: d.ID}).Result()
	_ = r.client.HSet(r.ctx, metaKey(d.ID), map[string]interface{}{"rating": fmt.Sprintf("%f", d.Rating), "online": strconv.FormatBool(d.Online), "updated": time.Now().Format(time.RFC3339)}).Err()
}

func (r *RedisGeo) Nearby(lat, lon float64, limit int) []models.Driver {
	res, err := r.client.GeoRadius(r.ctx, r.key, lon, lat, &redis.GeoRadiusQuery{Radius: 5000, Unit: "m", WithCoord: true, WithDist: true, Count: limit, Sort: "ASC"}).Result()
	if err != nil {
		return nil
	}
	out := make([]models.Driver, 0, len(res))
	for _, g := range res {
		d := models.Driver{ID: g.Name}
		// go-redis GeoLocation exposes Latitude and Longitude
		d.Loc.Lat = g.Latitude
		d.Loc.Lon = g.Longitude
		// try to fetch metadata
		if m, err := r.client.HGetAll(r.ctx, metaKey(g.Name)).Result(); err == nil {
			if v, ok := m["rating"]; ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					d.Rating = f
				}
			}
			if v, ok := m["online"]; ok {
				d.Online = (v == "true")
			}
		}
		out = append(out, d)
	}
	return out
}

func metaKey(id string) string { return "driver:meta:" + id }
