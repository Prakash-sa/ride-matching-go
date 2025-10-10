package dispatch

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/example/ride-matching/internal/models"
)

type PushDispatcher struct {
	Endpoint string // e.g. provider HTTP endpoint
	Client   *http.Client
	WS       *WSRegistry
}

func NewPushDispatcher(endpoint string, ws *WSRegistry) *PushDispatcher {
	return &PushDispatcher{Endpoint: endpoint, Client: &http.Client{Timeout: 3 * time.Second}, WS: ws}
}

func (p *PushDispatcher) Offer(rideID string, offer interface{}) error {
	// Try WS first if available
	if m, ok := offer.(map[string]interface{}); ok {
		if driverID, ok := m["driver_id"].(string); ok && p.WS != nil {
			// try to convert to known MatchOffer shape
			if eta, ok := m["eta"].(float64); ok {
				if cost, ok := m["cost"].(float64); ok {
					_ = p.WS.Offer(driverID, models.MatchOffer{DriverID: driverID, ETA: eta, Cost: cost})
					return nil
				}
			}
		}
	}
	// Fallback: post to Endpoint
	b, _ := json.Marshal(map[string]interface{}{"ride_id": rideID, "offer": offer})
	_, _ = p.Client.Post(p.Endpoint, "application/json", bytes.NewReader(b))
	return nil
}
