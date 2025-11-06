package repository

import (
	"database/sql"
	"fmt"

	"time"

	"github.com/google/uuid"
	"github.com/tullo/backend/internal/database"
	"github.com/tullo/backend/internal/models"
)

type ConversationRepository struct {
	db *database.DB
}

func NewConversationRepository(db *database.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

// Create creates a new conversation
func (r *ConversationRepository) Create(conversation *models.Conversation) error {
	query := `
		INSERT INTO conversations (id, is_group, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		conversation.ID,
		conversation.IsGroup,
		conversation.Name,
		conversation.CreatedAt,
		conversation.UpdatedAt,
	).Scan(&conversation.ID, &conversation.CreatedAt, &conversation.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	return nil
}

// GetByID retrieves a conversation by ID
func (r *ConversationRepository) GetByID(id uuid.UUID) (*models.Conversation, error) {
	query := `
		SELECT id, is_group, name, created_at, updated_at
		FROM conversations
		WHERE id = $1
	`

	conversation := &models.Conversation{}
	err := r.db.QueryRow(query, id).Scan(
		&conversation.ID,
		&conversation.IsGroup,
		&conversation.Name,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conversation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	return conversation, nil
}

// GetByUserID retrieves all conversations for a user
func (r *ConversationRepository) GetByUserID(userID uuid.UUID) ([]models.Conversation, error) {
	query := `
		SELECT c.id, c.is_group, c.name, c.created_at, c.updated_at
		FROM conversations c
		INNER JOIN conversation_members cm ON c.id = cm.conversation_id
		WHERE cm.user_id = $1
		ORDER BY c.updated_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}
	defer rows.Close()

	conversations := []models.Conversation{}
	for rows.Next() {
		var conv models.Conversation
		err := rows.Scan(
			&conv.ID,
			&conv.IsGroup,
			&conv.Name,
			&conv.CreatedAt,
			&conv.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// AddMember adds a member to a conversation
func (r *ConversationRepository) AddMember(member *models.ConversationMember) error {
	query := `
		INSERT INTO conversation_members (id, conversation_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (conversation_id, user_id) DO NOTHING
		RETURNING id, joined_at
	`

	err := r.db.QueryRow(
		query,
		member.ID,
		member.ConversationID,
		member.UserID,
		member.Role,
		member.JoinedAt,
	).Scan(&member.ID, &member.JoinedAt)

	if err == sql.ErrNoRows {
		// Member already exists
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	return nil
}

// RemoveMember removes a member from a conversation
func (r *ConversationRepository) RemoveMember(conversationID, userID uuid.UUID) error {
	query := `
		DELETE FROM conversation_members
		WHERE conversation_id = $1 AND user_id = $2
	`

	result, err := r.db.Exec(query, conversationID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("member not found")
	}

	return nil
}

// GetMembers retrieves all members of a conversation
func (r *ConversationRepository) GetMembers(conversationID uuid.UUID) ([]models.User, error) {
	query := `
		SELECT u.id, u.email, u.display_name, u.avatar_url, u.password_hash, u.created_at, u.updated_at
		FROM users u
		INNER JOIN conversation_members cm ON u.id = cm.user_id
		WHERE cm.conversation_id = $1
	`

	rows, err := r.db.Query(query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	defer rows.Close()

	members := []models.User{}
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.DisplayName,
			&user.AvatarURL,
			&user.PasswordHash,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, user)
	}

	return members, nil
}

// IsMember checks if a user is a member of a conversation
func (r *ConversationRepository) IsMember(conversationID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM conversation_members
			WHERE conversation_id = $1 AND user_id = $2
		)
	`

	var exists bool
	err := r.db.QueryRow(query, conversationID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return exists, nil
}

// GetOrCreateDirectConversation gets or creates a 1:1 conversation between two users
func (r *ConversationRepository) GetOrCreateDirectConversation(user1ID, user2ID uuid.UUID) (*models.Conversation, error) {
	// Check if conversation already exists
	query := `
		SELECT c.id, c.is_group, c.name, c.created_at, c.updated_at
		FROM conversations c
		INNER JOIN conversation_members cm1 ON c.id = cm1.conversation_id
		INNER JOIN conversation_members cm2 ON c.id = cm2.conversation_id
		WHERE c.is_group = false
		AND cm1.user_id = $1
		AND cm2.user_id = $2
		LIMIT 1
	`

	conversation := &models.Conversation{}
	err := r.db.QueryRow(query, user1ID, user2ID).Scan(
		&conversation.ID,
		&conversation.IsGroup,
		&conversation.Name,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)

	if err == nil {
		return conversation, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing conversation: %w", err)
	}

	// Create new conversation
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	conversation.ID = uuid.New()
	conversation.IsGroup = false

	_, err = tx.Exec(
		`INSERT INTO conversations (id, is_group, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())`,
		conversation.ID,
		conversation.IsGroup,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	// Add both members
	_, err = tx.Exec(
		`INSERT INTO conversation_members (id, conversation_id, user_id, role, joined_at) VALUES ($1, $2, $3, $4, NOW())`,
		uuid.New(), conversation.ID, user1ID, "member",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add first member: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO conversation_members (id, conversation_id, user_id, role, joined_at) VALUES ($1, $2, $3, $4, NOW())`,
		uuid.New(), conversation.ID, user2ID, "member",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add second member: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(conversation.ID)
}

// GetMemberRole returns the role of a member in a conversation (e.g., 'admin','moderator','member')
func (r *ConversationRepository) GetMemberRole(conversationID, userID uuid.UUID) (string, error) {
	query := `
		SELECT role FROM conversation_members WHERE conversation_id = $1 AND user_id = $2 LIMIT 1
	`
	var role string
	err := r.db.QueryRow(query, conversationID, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get member role: %w", err)
	}
	return role, nil
}

// AddModeration adds a moderation entry (mute/ban) for a user in a conversation
func (r *ConversationRepository) AddModeration(conversationID, userID uuid.UUID, action string, expiresAt *time.Time, reason string) error {
	query := `
		INSERT INTO conversation_moderations (id, conversation_id, user_id, action, expires_at, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (conversation_id, user_id, action) DO UPDATE SET expires_at = EXCLUDED.expires_at, reason = EXCLUDED.reason, created_at = NOW()
	`
	_, err := r.db.Exec(query, uuid.New(), conversationID, userID, action, expiresAt, reason)
	if err != nil {
		return fmt.Errorf("failed to add moderation: %w", err)
	}
	return nil
}

// RemoveModeration removes a moderation entry
func (r *ConversationRepository) RemoveModeration(conversationID, userID uuid.UUID, action string) error {
	query := `DELETE FROM conversation_moderations WHERE conversation_id = $1 AND user_id = $2 AND action = $3`
	_, err := r.db.Exec(query, conversationID, userID, action)
	if err != nil {
		return fmt.Errorf("failed to remove moderation: %w", err)
	}
	return nil
}

// IsUserMutedOrBanned checks if a user is currently muted or banned in a conversation
func (r *ConversationRepository) IsUserMutedOrBanned(conversationID, userID uuid.UUID) (muted bool, banned bool, err error) {
	query := `
		SELECT action, expires_at FROM conversation_moderations
		WHERE conversation_id = $1 AND user_id = $2
	`
	rows, err := r.db.Query(query, conversationID, userID)
	if err != nil {
		return false, false, fmt.Errorf("failed to check moderation: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	for rows.Next() {
		var action string
		var expiresAt sql.NullTime
		if err := rows.Scan(&action, &expiresAt); err != nil {
			return false, false, fmt.Errorf("failed to scan moderation: %w", err)
		}
		if expiresAt.Valid && expiresAt.Time.Before(now) {
			// expired; skip
			continue
		}
		if action == "mute" {
			muted = true
		}
		if action == "ban" {
			banned = true
		}
	}

	return muted, banned, nil
}

// UpdateMemberRole sets role for an existing member or inserts the member with given role
func (r *ConversationRepository) UpdateMemberRole(conversationID, userID uuid.UUID, role string) error {
	// try update
	res, err := r.db.Exec(`UPDATE conversation_members SET role = $1 WHERE conversation_id = $2 AND user_id = $3`, role, conversationID, userID)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows > 0 {
		return nil
	}

	// insert if not exists
	_, err = r.db.Exec(`INSERT INTO conversation_members (id, conversation_id, user_id, role, joined_at) VALUES ($1,$2,$3,$4,NOW()) ON CONFLICT DO NOTHING`, uuid.New(), conversationID, userID, role)
	if err != nil {
		return fmt.Errorf("failed to insert member role: %w", err)
	}
	return nil
}
