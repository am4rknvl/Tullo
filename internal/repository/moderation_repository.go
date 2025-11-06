package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/tullo/backend/internal/database"
	"github.com/tullo/backend/internal/models"
)

type ModerationRepository struct {
	db *database.DB
}

func NewModerationRepository(db *database.DB) *ModerationRepository {
	return &ModerationRepository{db: db}
}

// AddBannedWord adds a banned word for a conversation
func (r *ModerationRepository) AddBannedWord(conversationID uuid.UUID, word string) error {
	query := `INSERT INTO channel_banned_words (id, conversation_id, word, created_at) VALUES ($1,$2,$3,NOW()) ON CONFLICT (conversation_id, word) DO NOTHING`
	_, err := r.db.Exec(query, uuid.New(), conversationID, word)
	if err != nil {
		return fmt.Errorf("failed to add banned word: %w", err)
	}
	return nil
}

func (r *ModerationRepository) RemoveBannedWord(conversationID uuid.UUID, word string) error {
	query := `DELETE FROM channel_banned_words WHERE conversation_id = $1 AND word = $2`
	_, err := r.db.Exec(query, conversationID, word)
	if err != nil {
		return fmt.Errorf("failed to remove banned word: %w", err)
	}
	return nil
}

func (r *ModerationRepository) GetBannedWords(conversationID uuid.UUID) ([]models.BannedWord, error) {
	query := `SELECT id, conversation_id, word, created_at FROM channel_banned_words WHERE conversation_id = $1`
	rows, err := r.db.Query(query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query banned words: %w", err)
	}
	defer rows.Close()

	res := []models.BannedWord{}
	for rows.Next() {
		var b models.BannedWord
		if err := rows.Scan(&b.ID, &b.ConversationID, &b.Word, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan banned word: %w", err)
		}
		res = append(res, b)
	}
	return res, nil
}

// AddLog records a moderation action
func (r *ModerationRepository) AddLog(log *models.ModerationLog) error {
	meta := sql.NullString{}
	if log.Metadata != nil {
		if b, err := json.Marshal(log.Metadata); err == nil {
			meta = sql.NullString{String: string(b), Valid: true}
		}
	}

	query := `INSERT INTO moderation_logs (id, conversation_id, message_id, action, moderator_id, target_user_id, reason, metadata, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW()) RETURNING id, created_at`
	if _, err := r.db.Exec(query, log.ID, log.ConversationID, log.MessageID, log.Action, log.ModeratorID, log.TargetUserID, log.Reason, meta); err != nil {
		return fmt.Errorf("failed to insert moderation log: %w", err)
	}
	return nil
}

func (r *ModerationRepository) GetLogsByConversation(conversationID uuid.UUID, limit int) ([]models.ModerationLog, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, conversation_id, message_id, action, moderator_id, target_user_id, reason, metadata, created_at FROM moderation_logs WHERE conversation_id = $1 ORDER BY created_at DESC LIMIT $2`
	rows, err := r.db.Query(query, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query moderation logs: %w", err)
	}
	defer rows.Close()

	res := []models.ModerationLog{}
	for rows.Next() {
		var m models.ModerationLog
		var meta sql.NullString
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.MessageID, &m.Action, &m.ModeratorID, &m.TargetUserID, &m.Reason, &meta, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan moderation log: %w", err)
		}
		if meta.Valid {
			var mm map[string]any
			_ = json.Unmarshal([]byte(meta.String), &mm)
			m.Metadata = mm
		}
		res = append(res, m)
	}
	return res, nil
}
