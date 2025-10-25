package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/tullo/backend/internal/database"
	"github.com/tullo/backend/internal/models"
)

type MessageRepository struct {
	db *database.DB
}

func NewMessageRepository(db *database.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create creates a new message
func (r *MessageRepository) Create(message *models.Message) error {
	query := `
		INSERT INTO messages (id, conversation_id, sender_id, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		message.ID,
		message.ConversationID,
		message.SenderID,
		message.Body,
		message.CreatedAt,
		message.UpdatedAt,
	).Scan(&message.ID, &message.CreatedAt, &message.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

// GetByID retrieves a message by ID
func (r *MessageRepository) GetByID(id uuid.UUID) (*models.Message, error) {
	query := `
		SELECT id, conversation_id, sender_id, body, created_at, updated_at
		FROM messages
		WHERE id = $1
	`

	message := &models.Message{}
	err := r.db.QueryRow(query, id).Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.Body,
		&message.CreatedAt,
		&message.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return message, nil
}

// GetByConversationID retrieves messages for a conversation with pagination
func (r *MessageRepository) GetByConversationID(conversationID uuid.UUID, limit, offset int) ([]models.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT m.id, m.conversation_id, m.sender_id, m.body, m.created_at, m.updated_at,
		       u.id, u.email, u.display_name, u.avatar_url, u.password_hash, u.created_at, u.updated_at
		FROM messages m
		INNER JOIN users u ON m.sender_id = u.id
		WHERE m.conversation_id = $1
		ORDER BY m.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, conversationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	messages := []models.Message{}
	for rows.Next() {
		var msg models.Message
		var sender models.User

		err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.SenderID,
			&msg.Body,
			&msg.CreatedAt,
			&msg.UpdatedAt,
			&sender.ID,
			&sender.Email,
			&sender.DisplayName,
			&sender.AvatarURL,
			&sender.PasswordHash,
			&sender.CreatedAt,
			&sender.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.Sender = &sender
		messages = append(messages, msg)
	}

	return messages, nil
}

// MarkAsRead marks a message as read by a user
func (r *MessageRepository) MarkAsRead(messageID, userID uuid.UUID) error {
	query := `
		INSERT INTO message_reads (id, message_id, user_id, read_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (message_id, user_id) DO NOTHING
	`

	_, err := r.db.Exec(query, uuid.New(), messageID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark message as read: %w", err)
	}

	return nil
}

// GetReadReceipts retrieves read receipts for a message
func (r *MessageRepository) GetReadReceipts(messageID uuid.UUID) ([]models.MessageRead, error) {
	query := `
		SELECT id, message_id, user_id, read_at
		FROM message_reads
		WHERE message_id = $1
	`

	rows, err := r.db.Query(query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get read receipts: %w", err)
	}
	defer rows.Close()

	receipts := []models.MessageRead{}
	for rows.Next() {
		var receipt models.MessageRead
		err := rows.Scan(
			&receipt.ID,
			&receipt.MessageID,
			&receipt.UserID,
			&receipt.ReadAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan read receipt: %w", err)
		}
		receipts = append(receipts, receipt)
	}

	return receipts, nil
}

// GetUnreadCount gets the number of unread messages for a user in a conversation
func (r *MessageRepository) GetUnreadCount(conversationID, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM messages m
		LEFT JOIN message_reads mr ON m.id = mr.message_id AND mr.user_id = $2
		WHERE m.conversation_id = $1
		AND m.sender_id != $2
		AND mr.id IS NULL
	`

	var count int
	err := r.db.QueryRow(query, conversationID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

// Delete deletes a message
func (r *MessageRepository) Delete(id uuid.UUID) error {
	query := `DELETE FROM messages WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("message not found")
	}

	return nil
}
