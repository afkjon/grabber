package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/afkjon/grabber/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DATABASE_URL is the connection string to the database
const DATABASE_URL = "postgres://postgres:pg@localhost:5432/noodle_strainer?sslmode=disable"

// pool is the connection pool to the database
var pool *pgxpool.Pool

// Connect creates a connection pool to the database
func Connect() error {
	if pool != nil {
		return nil
	}

	var err error
	pool, err = pgxpool.New(context.Background(), DATABASE_URL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}

	// Apply migrations
	_, err = pool.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS migrations (id SERIAL PRIMARY KEY, name TEXT NOT NULL, applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW())")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Migrations failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Connected to database")
	return nil
}

// GetPendingJobs returns a pending job from the database
func GetPendingJobs() ([]any, error) {
	if pool == nil {
		return nil, fmt.Errorf("not connected to database")
	}
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "acquire failed: %v\n", err)
		os.Exit(1)
	}
	defer conn.Release()

	rows, err := conn.Query(context.Background(), "SELECT * FROM jobs WHERE status = 'pending'")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}

	return rows.Values()
}

// InsertShops inserts a list of shops into the database
func InsertShops(shopList []model.Shop) error {
	if shopList == nil {
		return fmt.Errorf("no shops to insert")
	}
	if pool == nil {
		return fmt.Errorf("not connected to database")
	}
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "acquire failed: %v\n", err)
		os.Exit(1)
	}
	defer conn.Release()

	for _, shop := range shopList {
		fmt.Printf("Inserting shop: %v\n", shop)
		_, err = conn.Exec(
			context.Background(),
			"INSERT INTO shops (name, address, tabelog_url, prefecture, price, station, station_distance) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			shop.Name, shop.Address, shop.TabelogURL, shop.Prefecture, shop.Price, shop.Station, shop.StationDistance,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed at inserting shop: %[1]v %[2]v\n", shop, err)
		}
	}

	return nil
}

// UpdateShop updates a shop in the database
// UpdateShop uses the tabelog_url as the unique identifier
func UpdateShop(shop model.Shop) error {
	if pool == nil {
		return fmt.Errorf("not connected to database")
	}
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "acquire failed: %v\n", err)
		os.Exit(1)
	}
	defer conn.Release()

	_, err = conn.Exec(
		context.Background(),
		"UPDATE shops SET name = $1, address = $2, prefecture = $3, price = $4, station = $5, station_distance = $6 WHERE tabelog_url = $7",
		shop.Name, shop.Address, shop.Prefecture, shop.Price, shop.Station, shop.StationDistance, shop.TabelogURL,
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed at updating shop: %[1]v %[2]v\n", shop, err)
		return err
	}
	fmt.Printf("Updated shop: %v\n", shop)

	return nil
}

// Close closes the connection pool
func Close() error {
	if pool == nil {
		return fmt.Errorf("not connected to database")
	}
	pool.Close()
	return nil
}

// GetShops returns all shops from the database
func GetShopsPendingGeocoding() ([]model.Shop, error) {
	if pool == nil {
		return nil, fmt.Errorf("not connected to database")
	}
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "acquire failed: %v\n", err)
		os.Exit(1)
	}
	defer conn.Release()

	rows, err := conn.Query(context.Background(), "SELECT id, address, is_geocoded FROM shops WHERE is_geocoded = FALSE LIMIT 1000")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}

	shops := []model.Shop{}
	for rows.Next() {
		var shop model.Shop
		err = rows.Scan(&shop.ID, &shop.Address, &shop.IsGeocoded)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
			os.Exit(1)
		}
		shops = append(shops, shop)
	}

	return shops, nil
}

// UpdateLocation updates the database with geocoded data
func UpdateLocation(id int, geoData *model.GoogleGeocodeResponseResultLocation, fullResponse json.RawMessage) error {
	if pool == nil {
		return fmt.Errorf("not connected to database")
	}

	conn, err := pool.Acquire(context.Background())
	if err != nil {
		return fmt.Errorf("acquire failed: %v\n", err)
	}
	defer conn.Release()

	// Start a transaction
	tx, err := conn.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background()) // Rollback on error

	// Prepare the statement
	_, err = tx.Prepare(context.Background(), "update_shop",
		`
		UPDATE shops
		SET latitude = $1,
			longitude = $2,
			location_type = $3,
			full_api_response = $4,
			is_geocoded = TRUE
		WHERE id = $5
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}

	// Execute the update
	_, err = tx.Exec(context.Background(), "update_shop", geoData.Lat, geoData.Lng, geoData.LocationType, fullResponse, id)
	if err != nil {
		return fmt.Errorf("failed to execute update for ID %d: %w", id, err)
	}

	return tx.Commit(context.Background()) // Commit the transaction
}
