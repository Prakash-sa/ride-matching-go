package eta

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/example/ride-matching/internal/models"
)

// OSRMClient performs route/eta lookups against an OSRM HTTP server.
type OSRMClient struct {
	Endpoint string
	Client   *http.Client
}

func NewOSRMClient(endpoint string) *OSRMClient {
	return &OSRMClient{Endpoint: endpoint, Client: &http.Client{Timeout: 2 * time.Second}}
}

// EstimateSeconds queries OSRM /route between points and returns duration in seconds.
func (o *OSRMClient) EstimateSeconds(from models.Coord, to models.Coord) (float64, error) {
	// OSRM route query: /route/v1/driving/{lon1},{lat1};{lon2},{lat2}?overview=false
	url := fmt.Sprintf("%s/route/v1/driving/%.6f,%.6f;%.6f,%.6f?overview=false", o.Endpoint, from.Lon, from.Lat, to.Lon, to.Lat)
	resp, err := o.Client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var out struct {
		Routes []struct {
			Duration float64 `json:"duration"`
		} `json:"routes"`
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	if out.Code != "Ok" || len(out.Routes) == 0 {
		return 0, fmt.Errorf("osrm no route: %v", out.Code)
	}
	return out.Routes[0].Duration, nil
}
