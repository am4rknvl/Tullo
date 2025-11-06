package models

import (
	"time"

	"github.com/google/uuid"
)

type Stream struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	ChannelID uuid.UUID  `json:"channel_id" db:"channel_id"`
	Status    string     `json:"status" db:"status"` // offline, live, ended
	IngestURL *string    `json:"ingest_url,omitempty" db:"ingest_url"`
	HLSURL    *string    `json:"hls_url,omitempty" db:"hls_url"`
	StreamKey *string    `json:"stream_key,omitempty" db:"stream_key"`
	StartedAt *time.Time `json:"started_at,omitempty" db:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}
