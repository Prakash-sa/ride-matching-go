-- simple rides table
CREATE TABLE IF NOT EXISTS rides (
  id TEXT PRIMARY KEY,
  rider_id TEXT NOT NULL,
  driver_id TEXT,
  origin_lat DOUBLE PRECISION,
  origin_lon DOUBLE PRECISION,
  dest_lat DOUBLE PRECISION,
  dest_lon DOUBLE PRECISION,
  status TEXT,
  created_at TIMESTAMP WITH TIME ZONE,
  updated_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_rides_status ON rides(status);
