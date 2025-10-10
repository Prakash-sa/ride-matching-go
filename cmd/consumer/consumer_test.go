package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/ride-matching/internal/models"
	"github.com/redis/go-redis/v9"
)

// fakeUpdater implements RedisUpdater for tests
type fakeUpdater struct {
	failGeo  int // number of times to fail GeoAdd before succeeding
	failH    int // number of times to fail HSet before succeeding
	geoCalls int
	hCalls   int
}

func (f *fakeUpdater) GeoAdd(ctx context.Context, key string, loc *redis.GeoLocation) error {
	f.geoCalls++
	if f.geoCalls <= f.failGeo {
		return errors.New("geo fail")
	}
	return nil
}

func (f *fakeUpdater) HSet(ctx context.Context, key string, values map[string]interface{}) error {
	f.hCalls++
	if f.hCalls <= f.failH {
		return errors.New("hset fail")
	}
	return nil
}

func TestUpdateRedisWithRetry_SucceedsAfterRetries(t *testing.T) {
	f := &fakeUpdater{failGeo: 1, failH: 1}
	d := &models.Driver{ID: "d1", Loc: models.Coord{Lat: 1, Lon: 2}, Rating: 4.5, Online: true}
	ctx := context.Background()
	start := time.Now()
	if err := updateRedisWithRetry(ctx, f, d, 3, 10*time.Millisecond); err != nil {
		t.Fatalf("expected success, got err=%v", err)
	}
	if f.geoCalls < 2 || f.hCalls < 2 {
		t.Fatalf("expected retries, got geo=%d h=%d", f.geoCalls, f.hCalls)
	}
	if time.Since(start) < 10*time.Millisecond {
		t.Fatalf("expected at least one backoff")
	}
}

func TestUpdateRedisWithRetry_FailsWhenExhausted(t *testing.T) {
	f := &fakeUpdater{failGeo: 5, failH: 0}
	d := &models.Driver{ID: "d1", Loc: models.Coord{Lat: 1, Lon: 2}, Rating: 4.5, Online: true}
	ctx := context.Background()
	if err := updateRedisWithRetry(ctx, f, d, 3, 5*time.Millisecond); err == nil {
		t.Fatalf("expected error after retries")
	}
}
