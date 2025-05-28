package model

import (
	"time"

	_ "github.com/jackc/pgx/v5/pgtype"
)

type Job struct {
	ID         string    `db:"job_id"`
	Status     string    `db:"status"`
	Parameters string    `db:"parameters"`
	CreatedAt  time.Time `db:"created_at"`
}

type Shop struct {
	ID              int `db:"shop_id"`
	Name            string
	Address         string
	TabelogURL      string `db:"link"`
	Station         string
	StationDistance string
	Price           string
	Prefecture      string
	JobID           string `db:"job_id"`
	IsGeocoded      bool   `db:"is_geocoded"`
}

// GoogleGeocodeResponse structures the relevant parts of the Google API response
type GoogleGeocodeResponse struct {
	Results []struct {
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			LocationType string `json:"location_type"`
		} `json:"geometry"`
		PlaceID string   `json:"place_id"`
		Types   []string `json:"types"`
	} `json:"results"`
	Status string `json:"status"`
}

// Create a custom struct to return just the core geo data
type GoogleGeocodeResponseResultLocation struct {
	Lat          float64
	Lng          float64
	LocationType string
}
