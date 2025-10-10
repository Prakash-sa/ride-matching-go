package matcher

import (
	"sort"
	"time"

	"github.com/example/ride-matching/internal/eta"
	"github.com/example/ride-matching/internal/models"
	"github.com/example/ride-matching/internal/observability"
	"github.com/example/ride-matching/internal/storage"
)

type Geo interface {
	Nearby(lat, lon float64, limit int) []models.Driver
}

type Dispatcher interface {
	Offer(rideID string, offer models.MatchOffer) error
}

type Service struct {
	Geo             Geo
	Dispatch        Dispatcher
	Store           storage.TripStore
	DefaultSpeedMps float64
	TopN            int
	ETAClient       eta.Client // optional OSRM client
	ETACache        *eta.Cache // optional ETA cache
}

func (s *Service) Match(rideID string, req models.RideRequest) (models.MatchOffer, bool) {
	if s.TopN <= 0 {
		s.TopN = 10
	}
	cands := s.Geo.Nearby(req.Origin.Lat, req.Origin.Lon, s.TopN)
	if len(cands) == 0 {
		return models.MatchOffer{}, false
	}
	type scored struct {
		d      models.Driver
		etaSec float64
		cost   float64
	}
	scoredList := make([]scored, 0, len(cands))
	for _, d := range cands {
		var etaSec float64
		if s.ETACache != nil {
			if v, ok := s.ETACache.Get(d.Loc, req.Origin); ok {
				etaSec = v
			}
		}
		if etaSec == 0 {
			if s.ETAClient != nil {
				if v, err := s.ETAClient.EstimateSeconds(d.Loc, req.Origin); err == nil {
					etaSec = v
					if s.ETACache != nil {
						s.ETACache.Set(d.Loc, req.Origin, etaSec)
					}
				} else {
					// fallback to naive estimator
					etaSec = eta.EstimateSeconds(d.Loc, req.Origin, s.DefaultSpeedMps)
				}
			} else {
				etaSec = eta.EstimateSeconds(d.Loc, req.Origin, s.DefaultSpeedMps)
			}
		}
		cost := etaSec + 30.0*(5.0-d.Rating) // cost = w1*eta + w2*(5 - rating)
		scoredList = append(scoredList, scored{d, etaSec, cost})
	}
	sort.Slice(scoredList, func(i, j int) bool { return scoredList[i].cost < scoredList[j].cost })

	best := scoredList[0]
	offer := models.MatchOffer{DriverID: best.d.ID, ETA: best.etaSec, Cost: best.cost}
	_ = s.Dispatch.Offer(rideID, offer) // best-effort for this demo
	observability.MatchesTotal.Inc()
	r := &models.Ride{
		ID:          rideID,
		RiderID:     req.RiderID,
		DriverID:    best.d.ID,
		Origin:      req.Origin,
		Destination: req.Destination,
		Status:      "matched",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	_ = s.Store.SaveRide(r)
	return offer, true
}
