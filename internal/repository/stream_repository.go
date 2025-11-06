package repository

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tullo/backend/internal/database"
	"github.com/tullo/backend/internal/models"
)

type StreamRepository struct {
	db *database.DB
}

func NewStreamRepository(db *database.DB) *StreamRepository {
	return &StreamRepository{db: db}
}

func (r *StreamRepository) Create(s *models.Stream) error {
	query := `
        INSERT INTO streams (id, channel_id, status, ingest_url, hls_url, stream_key, started_at, ended_at, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
        RETURNING id, created_at, updated_at
    `
	err := r.db.QueryRow(query,
		s.ID,
		s.ChannelID,
		s.Status,
		s.IngestURL,
		s.HLSURL,
		s.StreamKey,
		s.StartedAt,
		s.EndedAt,
		s.CreatedAt,
		s.UpdatedAt,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	return nil
}

func (r *StreamRepository) UpdateStatus(id uuid.UUID, status string) error {
	query := `UPDATE streams SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update stream status: %w", err)
	}
	return nil
}

func (r *StreamRepository) GetByChannel(channelID uuid.UUID) (*models.Stream, error) {
	query := `
        SELECT id, channel_id, status, ingest_url, hls_url, stream_key, started_at, ended_at, created_at, updated_at
        FROM streams WHERE channel_id = $1 ORDER BY created_at DESC LIMIT 1
    `
	s := &models.Stream{}
	err := r.db.QueryRow(query, channelID).Scan(
		&s.ID,
		&s.ChannelID,
		&s.Status,
		&s.IngestURL,
		&s.HLSURL,
		&s.StreamKey,
		&s.StartedAt,
		&s.EndedAt,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream: %w", err)
	}
	return s, nil
}

// GetActiveStreams returns streams currently marked as 'live'
func (r *StreamRepository) GetActiveStreams(limit int) ([]models.Stream, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
        SELECT id, channel_id, status, ingest_url, hls_url, stream_key, started_at, ended_at, created_at, updated_at
        FROM streams WHERE status = 'live' ORDER BY started_at DESC LIMIT $1
    `
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get active streams: %w", err)
	}
	defer rows.Close()

	var out []models.Stream
	for rows.Next() {
		var s models.Stream
		if err := rows.Scan(&s.ID, &s.ChannelID, &s.Status, &s.IngestURL, &s.HLSURL, &s.StreamKey, &s.StartedAt, &s.EndedAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan stream: %w", err)
		}
		out = append(out, s)
	}
	return out, nil
}

// EndStream sets stream status to ended and records ended_at
func (r *StreamRepository) EndStream(id uuid.UUID, endedAt time.Time) error {
	query := `UPDATE streams SET status = 'ended', ended_at = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, endedAt, id)
	if err != nil {
		return fmt.Errorf("failed to end stream: %w", err)
	}
	return nil
}
