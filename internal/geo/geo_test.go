package geo

import "testing"

func TestHaversineZero(t *testing.T) {
    d := Haversine(0,0,0,0)
    if d != 0 {
        t.Fatalf("expected 0, got %f", d)
    }
}
