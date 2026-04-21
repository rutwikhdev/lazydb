package db

// Driver package creates a new database driver for particular database and fetches the tables list
// table list is displayed as the start state of the tui
import (
	"database/sql"
	"fmt"
	"log"
)

type Database struct {
	DB *sql.DB
}

func NewDB(dbType string, dbPath string) (*Database, error) {
	fmt.Printf("Connecting to %s", dbType)
	db, err := sql.Open(dbType, dbPath)
	if err != nil {
		log.Fatalf("Failed to open connection for %s with path %s, error %s", dbType, dbPath, err.Error())
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{DB: db}

	return database, nil
}
