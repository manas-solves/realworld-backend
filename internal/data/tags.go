package data

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TagStore struct {
	db      *pgxpool.Pool
	timeout time.Duration
}

// GetAll retrieves all tags from the database.
func (s *TagStore) GetAll() ([]string, error) {
	query := `SELECT ARRAY_AGG(tag ORDER BY tag) FROM tags`

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	var tags []string
	err := s.db.QueryRow(ctx, query).Scan(&tags)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []string{}, nil // Return empty slice if no tags exist
		}
		return nil, err
	}

	// Handle case where no tags exist (ARRAY_AGG returns NULL)
	if tags == nil {
		return []string{}, nil
	}

	return tags, nil
}
