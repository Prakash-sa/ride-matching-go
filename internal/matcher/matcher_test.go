package matcher

import (
    "testing"
    "github.com/example/ride-matching/internal/models"
)

type fakeGeo struct{ drivers []models.Driver }
func (f *fakeGeo) Nearby(lat, lon float64, limit int) []models.Driver { return f.drivers }

type nopDisp struct{}
func (n *nopDisp) Offer(rideID string, offer models.MatchOffer) error { return nil }

type memStore struct{ r *models.Ride }
func (m *memStore) SaveRide(r *models.Ride) error { m.r = r; return nil }
func (m *memStore) UpdateRide(r *models.Ride) error { m.r = r; return nil }

func TestChooseHigherRatingIfETAEqual(t *testing.T) {
    g := &fakeGeo{drivers: []models.Driver{
        {ID:"A", Loc: models.Coord{Lat:0,Lon:0}, Rating:4.0, Online:true},
        {ID:"B", Loc: models.Coord{Lat:0,Lon:0}, Rating:5.0, Online:true},
    }}
    s := &Service{Geo:g, Dispatch:&nopDisp{}, Store:&memStore{}, DefaultSpeedMps:10, TopN:2}
    req := models.RideRequest{RiderID:"r1", Origin:models.Coord{Lat:0,Lon:0}, Destination:models.Coord{Lat:0.1,Lon:0.1}}
    offer, ok := s.Match("ride1", req)
    if !ok { t.Fatal("no match") }
    if offer.DriverID != "B" {
        t.Fatalf("expected B, got %s", offer.DriverID)
    }
}
