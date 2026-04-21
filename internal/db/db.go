package db

// Driver package creates a new database driver for particular database and fetches the tables list
// table list is displayed as the start state of the tui
import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

type Database struct {
	DB   *sql.DB
	Type string
}

func NewDB(dbType string, connString string) (*Database, error) {
	fmt.Printf("Connecting to %s", dbType)
	db, err := sql.Open(dbType, connString)
	if err != nil {
		log.Fatalf("Failed to open connection for %s with path %s, error %s", dbType, connString, err.Error())
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{DB: db, Type: dbType}

	return database, nil
}

func (db *Database) FetchTables() []string {
	query, ok := Tables[db.Type]
	if !ok || len(query) == 0 {
		log.Fatal("%s database not supported", db.Type)
	}

	rows, err := db.DB.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	tables := []string{}

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Fatal(err)
		}
		tables = append(tables, tableName)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return tables
}
