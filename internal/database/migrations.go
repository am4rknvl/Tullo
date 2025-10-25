package database

import (
	"database/sql"
	"fmt"
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

			CREATE INDEX idx_users_email ON users(email);
		`,
		Down: `
			DROP TABLE IF EXISTS users;
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

			CREATE INDEX idx_conversations_created_at ON conversations(created_at DESC);
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

			CREATE INDEX idx_conversation_members_conversation ON conversation_members(conversation_id);
			CREATE INDEX idx_conversation_members_user ON conversation_members(user_id);
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

			CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at DESC);
			CREATE INDEX idx_messages_sender ON messages(sender_id);
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

			CREATE INDEX idx_message_reads_message ON message_reads(message_id);
			CREATE INDEX idx_message_reads_user ON message_reads(user_id);
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

	// Run pending migrations
	for _, migration := range Migrations {
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
