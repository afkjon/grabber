package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// urlExample := "postgres://username:password@localhost:5432/database_name"
const DATABASE_URL = "postgres://postgres:pg@localhost:5432/noodle_strainer?sslmode=disable"

var pool *pgxpool.Pool

// Connect is a function that creates a connection pool to the database
func Connect() error {
	if pool != nil {
		return nil
	}
	pool, err := pgxpool.New(context.Background(), DATABASE_URL)
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

	fmt.Print("Connected to database")
	return nil
}

// GetPendingJobs is a function that returns a pending job from the database
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
