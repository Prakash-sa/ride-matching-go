package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	"github.com/example/ride-matching/internal/models"
)

var (
	msgsConsumed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "consumer_messages_consumed_total",
		Help: "Total driver location messages consumed",
	})
	msgsInvalid = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "consumer_messages_invalid_total",
		Help: "Total invalid messages received",
	})
	redisUpdates = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "consumer_redis_updates_total",
		Help: "Total successful redis updates",
	})
	redisErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "consumer_redis_errors_total",
		Help: "Total redis errors",
	})
)

func init() {
	prometheus.MustRegister(msgsConsumed, msgsInvalid, redisUpdates, redisErrors)
}

func main() {
	// allow some flags for local runs
	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-addr", ":2112", "address to serve prometheus metrics on")
	flag.Parse()

	brokersEnv := os.Getenv("KAFKA_BROKERS")
	if brokersEnv == "" {
		brokersEnv = os.Getenv("KAFKA_BROKER")
	}
	brokers := []string{}
	if brokersEnv != "" {
		for _, b := range strings.Split(brokersEnv, ",") {
			if s := strings.TrimSpace(b); s != "" {
				brokers = append(brokers, s)
			}
		}
	} else {
		brokers = []string{"localhost:9092"}
	}

	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "driver-locations"
	}
	group := os.Getenv("KAFKA_GROUP")
	if group == "" {
		group = "ride-matching-consumer"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rc := redis.NewClient(&redis.Options{Addr: redisAddr})
	radapter := &redisAdapter{c: rc}

	// start metrics and health server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); w.Write([]byte("ok")) })
		mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
			// readiness: check redis connectivity
			if err := rc.Ping(r.Context()).Err(); err != nil {
				http.Error(w, "redis not ready", 503)
				return
			}
			w.WriteHeader(200)
			w.Write([]byte("ready"))
		})
		log.Printf("metrics/health listening on %s", metricsAddr)
		if err := http.ListenAndServe(metricsAddr, mux); err != nil {
			log.Printf("metrics server stopped: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	r := kafka.NewReader(kafka.ReaderConfig{Brokers: brokers, Topic: topic, GroupID: group, MinBytes: 10e3, MaxBytes: 10e6})
	defer func() {
		_ = r.Close()
		_ = rc.Close()
	}()

	log.Printf("consumer listening topic=%s brokers=%v group=%s", topic, brokers, group)

	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("shutting down consumer")
				return
			}
			log.Printf("kafka read error: %v; backing off %s", err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		// reset backoff on success
		backoff = time.Second

		msgsConsumed.Inc()

		var d models.Driver
		if err := json.Unmarshal(m.Value, &d); err != nil {
			msgsInvalid.Inc()
			log.Printf("invalid message: %v", err)
			continue
		}

		// Try updating Redis with retries and small backoff
		if err := updateRedisWithRetry(ctx, radapter, &d, 3, 200*time.Millisecond); err != nil {
			redisErrors.Inc()
			log.Printf("redis update failed for driver=%s: %v", d.ID, err)
			continue
		}
		redisUpdates.Inc()
	}
}

// RedisUpdater defines the small subset of redis operations we need for tests and production.
type RedisUpdater interface {
	GeoAdd(ctx context.Context, key string, loc *redis.GeoLocation) error
	HSet(ctx context.Context, key string, values map[string]interface{}) error
}

type redisAdapter struct{ c *redis.Client }

func (r *redisAdapter) GeoAdd(ctx context.Context, key string, loc *redis.GeoLocation) error {
	_, err := r.c.GeoAdd(ctx, key, loc).Result()
	return err
}

func (r *redisAdapter) HSet(ctx context.Context, key string, values map[string]interface{}) error {
	_, err := r.c.HSet(ctx, key, values).Result()
	return err
}

// updateRedisWithRetry updates redis using the RedisUpdater interface with retry/backoff.
func updateRedisWithRetry(ctx context.Context, rc RedisUpdater, d *models.Driver, attempts int, delay time.Duration) error {
	for i := 0; i < attempts; i++ {
		if err := rc.GeoAdd(ctx, "drivers_geo", &redis.GeoLocation{Longitude: d.Loc.Lon, Latitude: d.Loc.Lat, Name: d.ID}); err != nil {
			if i == attempts-1 {
				return err
			}
			time.Sleep(delay)
			delay *= 2
			continue
		}
		if err := rc.HSet(ctx, "driver:meta:"+d.ID, map[string]interface{}{"rating": d.Rating, "online": d.Online}); err != nil {
			if i == attempts-1 {
				return err
			}
			time.Sleep(delay)
			delay *= 2
			continue
		}
		return nil
	}
	return nil
}
