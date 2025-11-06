package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/tullo/backend/internal/database"
	"github.com/tullo/backend/internal/models"
)

type ChannelRepository struct {
	db *database.DB
}

func NewChannelRepository(db *database.DB) *ChannelRepository {
	return &ChannelRepository{db: db}
}

func (r *ChannelRepository) Create(channel *models.Channel) error {
	query := `
	INSERT INTO channels (id, owner_id, slug, title, description, language, tags, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
        RETURNING id, created_at, updated_at
    `
	err := r.db.QueryRow(query,
		channel.ID,
		channel.OwnerID,
		channel.Slug,
		channel.Title,
		channel.Description,
		channel.Language,
		pq.Array(channel.Tags),
		channel.CreatedAt,
		channel.UpdatedAt,
	).Scan(&channel.ID, &channel.CreatedAt, &channel.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create channel: %w", err)
	}
	return nil
}

func (r *ChannelRepository) GetBySlug(slug string) (*models.Channel, error) {
	query := `
	SELECT id, owner_id, slug, title, description, language, tags, created_at, updated_at
        FROM channels WHERE slug = $1
    `
	ch := &models.Channel{}
	var tags []string
	err := r.db.QueryRow(query, slug).Scan(
		&ch.ID,
		&ch.OwnerID,
		&ch.Slug,
		&ch.Title,
		&ch.Description,
		&ch.Language,
		pq.Array(&tags),
		&ch.CreatedAt,
		&ch.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	ch.Tags = tags
	return ch, nil
}

// GetOrCreateConversation returns the conversation id associated with a channel, creating one if missing
func (r *ChannelRepository) GetOrCreateConversation(channelID uuid.UUID) (uuid.UUID, error) {
	// Check if channel has conversation_id
	var convID sql.NullString
	err := r.db.QueryRow("SELECT conversation_id FROM channels WHERE id = $1", channelID).Scan(&convID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to query channel: %w", err)
	}
	if convID.Valid {
		id, err := uuid.Parse(convID.String)
		if err == nil {
			return id, nil
		}
	}

	// Create conversation and set it on channel in a transaction
	tx, err := r.db.Begin()
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	convIDNew := uuid.New()
	_, err = tx.Exec(`INSERT INTO conversations (id, is_group, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())`, convIDNew, true)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	_, err = tx.Exec(`UPDATE channels SET conversation_id = $1 WHERE id = $2`, convIDNew, channelID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to update channel: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("failed to commit: %w", err)
	}

	return convIDNew, nil
}

// AddFollower creates a follow record for a user on a channel
func (r *ChannelRepository) AddFollower(channelID, userID uuid.UUID) error {
	query := `INSERT INTO channel_follows (id, channel_id, user_id, created_at) VALUES ($1, $2, $3, NOW()) ON CONFLICT (channel_id, user_id) DO NOTHING`
	_, err := r.db.Exec(query, uuid.New(), channelID, userID)
	if err != nil {
		return fmt.Errorf("failed to add follower: %w", err)
	}
	return nil
}

// RemoveFollower removes a follow record
func (r *ChannelRepository) RemoveFollower(channelID, userID uuid.UUID) error {
	query := `DELETE FROM channel_follows WHERE channel_id = $1 AND user_id = $2`
	_, err := r.db.Exec(query, channelID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove follower: %w", err)
	}
	return nil
}

// IsFollower checks if a user follows a channel
func (r *ChannelRepository) IsFollower(channelID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM channel_follows WHERE channel_id = $1 AND user_id = $2)`
	var exists bool
	err := r.db.QueryRow(query, channelID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check follower: %w", err)
	}
	return exists, nil
}

// CountFollowers returns number of followers for a channel
func (r *ChannelRepository) CountFollowers(channelID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM channel_follows WHERE channel_id = $1`
	var cnt int
	err := r.db.QueryRow(query, channelID).Scan(&cnt)
	if err != nil {
		return 0, fmt.Errorf("failed to count followers: %w", err)
	}
	return cnt, nil
}
