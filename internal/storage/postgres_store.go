package storage

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"

	"github.com/example/ride-matching/internal/models"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	// quick ping
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresStore{db: db}, nil
}

func (p *PostgresStore) SaveRide(r *models.Ride) error {
	_, err := p.db.Exec(`INSERT INTO rides(id, rider_id, driver_id, origin_lat, origin_lon, dest_lat, dest_lon, status, created_at, updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		r.ID, r.RiderID, r.DriverID, r.Origin.Lat, r.Origin.Lon, r.Destination.Lat, r.Destination.Lon, r.Status, r.CreatedAt, r.UpdatedAt)
	return err
}

func (p *PostgresStore) UpdateRide(r *models.Ride) error {
	_, err := p.db.Exec(`UPDATE rides SET driver_id=$1, status=$2, updated_at=$3 WHERE id=$4`, r.DriverID, r.Status, time.Now(), r.ID)
	return err
}
