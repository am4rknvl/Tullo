package database

import (
	"database/sql"
	"fmt"
	"sort"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Up      string
	Down    string
}

// Migrations contains all database migrations
var Migrations = []Migration{
	{
		Version: 1,
		Up: `
			CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

			CREATE TABLE IF NOT EXISTS users (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				email VARCHAR(255) UNIQUE NOT NULL,
				display_name VARCHAR(255) NOT NULL,
				avatar_url TEXT,
				password_hash VARCHAR(255) NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
		`,
		Down: `
			DROP TABLE IF EXISTS users;
		`,
	},
	{
		Version: 11,
		Up: `
			CREATE TABLE IF NOT EXISTS channel_banned_words (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
				word VARCHAR(255) NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				UNIQUE(conversation_id, word)
			);

			CREATE TABLE IF NOT EXISTS moderation_logs (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				conversation_id UUID,
				message_id UUID,
				action VARCHAR(50) NOT NULL,
				moderator_id UUID,
				target_user_id UUID,
				reason TEXT,
				metadata JSONB,
				created_at TIMESTAMP NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_moderation_logs_conversation ON moderation_logs(conversation_id);
			CREATE INDEX IF NOT EXISTS idx_moderation_logs_target ON moderation_logs(target_user_id);
		`,
		Down: `
			DROP TABLE IF EXISTS channel_banned_words;
			DROP TABLE IF EXISTS moderation_logs;
		`,
	},
	{
		Version: 9,
		Up: `
			ALTER TABLE channels ADD COLUMN IF NOT EXISTS conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL;
		`,
		Down: `
			ALTER TABLE channels DROP COLUMN IF EXISTS conversation_id;
		`,
	},
	{
		Version: 7,
		Up: `
			CREATE TABLE IF NOT EXISTS channels (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				slug VARCHAR(255) UNIQUE NOT NULL,
				title VARCHAR(255) NOT NULL,
				description TEXT,
				language VARCHAR(50),
				tags TEXT[],
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_channels_owner ON channels(owner_id);
			CREATE INDEX IF NOT EXISTS idx_channels_slug ON channels(slug);
		`,
		Down: `
			DROP TABLE IF EXISTS channels;
		`,
	},
	{
		Version: 8,
		Up: `
			CREATE TABLE IF NOT EXISTS streams (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
				status VARCHAR(50) NOT NULL DEFAULT 'offline',
				ingest_url TEXT,
				hls_url TEXT,
				stream_key VARCHAR(255) UNIQUE,
				started_at TIMESTAMP,
				ended_at TIMESTAMP,
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_streams_channel ON streams(channel_id);
		`,
		Down: `
			DROP TABLE IF EXISTS streams;
		`,
	},
	{
		Version: 2,
		Up: `
			CREATE TABLE IF NOT EXISTS conversations (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				is_group BOOLEAN NOT NULL DEFAULT false,
				name VARCHAR(255),
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at DESC);
		`,
		Down: `
			DROP TABLE IF EXISTS conversations;
		`,
	},
	{
		Version: 3,
		Up: `
			CREATE TABLE IF NOT EXISTS conversation_members (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
				user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				role VARCHAR(50) NOT NULL DEFAULT 'member',
				joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
				UNIQUE(conversation_id, user_id)
			);

			CREATE INDEX IF NOT EXISTS idx_conversation_members_conversation ON conversation_members(conversation_id);
			CREATE INDEX IF NOT EXISTS idx_conversation_members_user ON conversation_members(user_id);
		`,
		Down: `
			DROP TABLE IF EXISTS conversation_members;
		`,
	},
	{
		Version: 4,
		Up: `
			CREATE TABLE IF NOT EXISTS messages (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
				sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				body TEXT NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMP NOT NULL DEFAULT NOW()
			);

			CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at DESC);
			CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id);
		`,
		Down: `
			DROP TABLE IF EXISTS messages;
		`,
	},
	{
		Version: 5,
		Up: `
			CREATE TABLE IF NOT EXISTS message_reads (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
				user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				read_at TIMESTAMP NOT NULL DEFAULT NOW(),
				UNIQUE(message_id, user_id)
			);

			CREATE INDEX IF NOT EXISTS idx_message_reads_message ON message_reads(message_id);
			CREATE INDEX IF NOT EXISTS idx_message_reads_user ON message_reads(user_id);
		`,
		Down: `
			DROP TABLE IF EXISTS message_reads;
		`,
	},
	{
		Version: 6,
		Up: `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version INT PRIMARY KEY,
				applied_at TIMESTAMP NOT NULL DEFAULT NOW()
			);
		`,
		Down: `
			DROP TABLE IF EXISTS schema_migrations;
		`,
	},
	{
		Version: 10,
		Up: `
			CREATE TABLE IF NOT EXISTS conversation_moderations (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
				user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				action VARCHAR(50) NOT NULL, -- 'mute' or 'ban'
				expires_at TIMESTAMP NULL,
				reason TEXT,
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				UNIQUE(conversation_id, user_id, action)
			);

			CREATE INDEX IF NOT EXISTS idx_conversation_moderations_conversation ON conversation_moderations(conversation_id);
			CREATE INDEX IF NOT EXISTS idx_conversation_moderations_user ON conversation_moderations(user_id);
		`,
		Down: `
			DROP TABLE IF EXISTS conversation_moderations;
		`,
	},
	{
		Version: 11,
		Up: `
			CREATE TABLE IF NOT EXISTS channel_follows (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
				user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				created_at TIMESTAMP NOT NULL DEFAULT NOW(),
				UNIQUE(channel_id, user_id)
			);

			CREATE INDEX IF NOT EXISTS idx_channel_follows_channel ON channel_follows(channel_id);
			CREATE INDEX IF NOT EXISTS idx_channel_follows_user ON channel_follows(user_id);
		`,
		Down: `
			DROP TABLE IF EXISTS channel_follows;
		`,
	},
}

// RunMigrations runs all pending migrations
func RunMigrations(db *sql.DB) error {
	// Ensure migrations table exists
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	// Get current version
	currentVersion, err := getCurrentVersion(db)
	if err != nil {
		return err
	}

	// Run pending migrations in ascending order by version
	sorted := make([]Migration, len(Migrations))
	copy(sorted, Migrations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Version < sorted[j].Version })

	// Run pending migrations
	for _, migration := range sorted {
		if migration.Version <= currentVersion {
			continue
		}

		fmt.Printf("Running migration %d...\n", migration.Version)

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		if _, err := tx.Exec(migration.Up); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to run migration %d: %w", migration.Version, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migration.Version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		fmt.Printf("Migration %d completed\n", migration.Version)
	}

	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func getCurrentVersion(db *sql.DB) (int, error) {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}
