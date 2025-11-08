package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/example/ride-matching/internal/config"
	"github.com/example/ride-matching/internal/dispatch"
	"github.com/example/ride-matching/internal/geo"
	"github.com/example/ride-matching/internal/ingest"
	"github.com/example/ride-matching/internal/matcher"
	"github.com/example/ride-matching/internal/models"
	"github.com/example/ride-matching/internal/observability"
	"github.com/example/ride-matching/internal/storage"
)

type Server struct {
	cfg    config.ServerConfig
	logger *slog.Logger

	Geo     geo.Geo
	Matcher *matcher.Service
	Store   storage.TripStore
	Kafka   *ingest.KafkaProducer
	WSReg   *dispatch.WSRegistry
	mux     *mux.Router
}

func NewServer(cfg config.ServerConfig, logger *slog.Logger) (*Server, error) {
	var ggeo geo.Geo
	if cfg.RedisAddr != "" {
		ggeo = geo.NewRedisGeo(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisGeoKey)
	} else {
		ggeo = geo.NewIndex()
	}

	var store storage.TripStore
	if cfg.PGDSN != "" {
		if ps, err := storage.NewPostgresStore(cfg.PGDSN); err == nil {
			store = ps
		} else {
			logger.Warn("postgres store init failed, falling back to memory store", "error", err)
		}
	}
	if store == nil {
		store = storage.NewMemoryStore()
	}

	var kp *ingest.KafkaProducer
	if len(cfg.KafkaBrokers) > 0 {
		kp = ingest.NewKafkaProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	}

	wsreg := dispatch.NewWSRegistry()

	m := &matcher.Service{Geo: ggeo, Dispatch: wsreg, Store: store, DefaultSpeedMps: cfg.DefaultSpeedMps, TopN: cfg.MatcherTopN}

	router := mux.NewRouter()
	s := &Server{
		cfg:     cfg,
		logger:  logger,
		Geo:     ggeo,
		Matcher: m,
		Store:   store,
		Kafka:   kp,
		WSReg:   wsreg,
		mux:     router,
	}
	s.routes()
	s.registerMiddleware()
	return s, nil
}

func (s *Server) routes() {
	s.mux.HandleFunc("/internal/driver/locations", s.handleDriverLocation).Methods("POST")
	s.mux.HandleFunc("/api/v1/rides/request", s.handleRideRequest).Methods("POST")
	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); w.Write([]byte("ok")) }).Methods("GET")
	s.mux.Handle("/metrics", promhttp.Handler())
	s.mux.HandleFunc("/ws/{driver_id}", s.handleWS)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) handleDriverLocation(w http.ResponseWriter, r *http.Request) {
	var d models.Driver
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	d.Online = true
	// publish to kafka if configured
	if s.Kafka != nil {
		_ = s.Kafka.PublishLocation(d)
	}
	// update geo store
	if up, ok := s.Geo.(interface{ Upsert(models.Driver) }); ok {
		up.Upsert(d)
	}
	// update metrics
	observability.DriversOnline.Inc()
	w.WriteHeader(204)
}

func (s *Server) handleRideRequest(w http.ResponseWriter, r *http.Request) {
	var rr models.RideRequest
	if err := json.NewDecoder(r.Body).Decode(&rr); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	rideID := newID()
	offer, ok := s.Matcher.Match(rideID, rr)
	if !ok {
		http.Error(w, "no drivers available", 503)
		return
	}
	observability.MatchesTotal.Inc()
	resp := map[string]any{"ride_id": rideID, "offer": offer}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

var upgrader = websocket.Upgrader{}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["driver_id"]
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "upgrade failed", 400)
		return
	}
	s.WSReg.Add(id, conn)
}

func newID() string { b := make([]byte, 8); _, _ = rand.Read(b); return hex.EncodeToString(b) }
