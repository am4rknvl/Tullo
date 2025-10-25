package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/tullo/backend/config"
	"github.com/tullo/backend/internal/database"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/migrate/main.go [up|down|status]")
		os.Exit(1)
	}

	command := os.Args[1]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	switch command {
	case "up":
		log.Println("Running migrations...")
		if err := database.RunMigrations(db); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migrations completed successfully")

	case "status":
		showMigrationStatus(db)

	case "down":
		log.Println("Rollback not implemented yet")
		// TODO: Implement rollback

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: up, down, status")
		os.Exit(1)
	}
}

func showMigrationStatus(db *sql.DB) {
	rows, err := db.Query("SELECT version, applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		log.Printf("No migrations found or table doesn't exist: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("\nApplied Migrations:")
	fmt.Println("-------------------")
	for rows.Next() {
		var version int
		var appliedAt string
		if err := rows.Scan(&version, &appliedAt); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		fmt.Printf("Version %d - Applied at: %s\n", version, appliedAt)
	}
}
