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
	ID      string `db:"shop_id"`
	Name    string
	Address string
	Link    string
	JobID   string `db:"job_id"`
}
