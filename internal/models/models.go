package models

import "time"

type Coord struct {
    Lat float64 `json:"lat"`
    Lon float64 `json:"lon"`
}

type RideRequest struct {
    RiderID     string `json:"rider_id"`
    Origin      Coord  `json:"origin"`
    Destination Coord  `json:"destination"`
}

type Driver struct {
    ID      string  `json:"id"`
    Loc     Coord   `json:"loc"`
    Rating  float64 `json:"rating"` // 0..5
    Online  bool    `json:"online"`
    Updated time.Time `json:"updated"`
}

type MatchOffer struct {
    DriverID string  `json:"driver_id"`
    ETA      float64 `json:"eta_seconds"`
    Cost     float64 `json:"cost"`
}

type MatchDecision struct {
    RideID   string `json:"ride_id"`
    DriverID string `json:"driver_id"`
    Accepted bool   `json:"accepted"`
}

type Ride struct {
    ID        string
    RiderID   string
    DriverID  string
    Origin    Coord
    Destination Coord
    Status    string // requested, matched, accepted, ongoing, completed, canceled
    CreatedAt time.Time
    UpdatedAt time.Time
}
