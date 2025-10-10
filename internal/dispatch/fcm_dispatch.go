package dispatch

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type HTTPPush interface {
	Offer(rideID string, payload interface{}) error
}

// FCMDispatcher posts JSON to FCM HTTPv1 endpoint using server key or oauth token.
type FCMDispatcher struct {
	Endpoint string
	Key      string
	Client   *http.Client
}

func NewFCMDispatcher(endpoint, key string) *FCMDispatcher {
	return &FCMDispatcher{Endpoint: endpoint, Key: key, Client: &http.Client{Timeout: 3 * time.Second}}
}

func (f *FCMDispatcher) Offer(rideID string, payload interface{}) error {
	body := map[string]interface{}{"message": map[string]interface{}{"token": "", "data": map[string]interface{}{"ride_id": rideID, "offer": payload}}}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", f.Endpoint, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if f.Key != "" {
		req.Header.Set("Authorization", "Bearer "+f.Key)
	}
	_, _ = f.Client.Do(req)
	return nil
}
