package dispatch

import (
    "encoding/json"
    "log"
    "net/http"
    "time"

    "github.com/example/ride-matching/internal/models"
)

// HTTP dispatcher simulates notifying a driver app backend.
type HTTPDispatcher struct {
    Endpoint string
    Client   *http.Client
}

func (d *HTTPDispatcher) Offer(rideID string, offer models.MatchOffer) error {
    if d.Client == nil {
        d.Client = &http.Client{Timeout: 2 * time.Second}
    }
    payload := map[string]any{"ride_id": rideID, "offer": offer}
    b, _ := json.Marshal(payload)
    req, _ := http.NewRequest("POST", d.Endpoint,  http.NoBody)
    // For the demo, just log instead of real HTTP post.
    log.Printf("[dispatch] ride=%s offer=%s body=%s", rideID, offer.DriverID, string(b))
    _ = req
    return nil
}
