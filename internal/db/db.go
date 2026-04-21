package db

// Driver package creates a new database driver for particular database and fetches the tables list
// table list is displayed as the start state of the tui
import (
	"database/sql"
	"fmt"
	"log"
	"strings"

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
		fmt.Printf("%s database not supported", db.Type)
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

func (db *Database) GetColumns(table string) []string {
	rows, err := db.DB.Query(fmt.Sprintf("PRAGMA table_info(%s);", table))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var columns []string

	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue any // interface{}
			pk        int
		)

		err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		if err != nil {
			log.Fatal(err)
		}

		columns = append(columns, name)
	}

	return columns
}

func (db *Database) GetRows(table string, columns []string) ([][]string, error) {
	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns provided")
	}

	query := fmt.Sprintf("SELECT %s FROM %s",
		strings.Join(columns, ", "),
		table,
	)

	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results [][]string

	for rows.Next() {
		// Prepare a slice for raw values
		rawValues := make([]any, len(columns)) // interface{} -> any
		valuePtrs := make([]any, len(columns)) // interface{} -> any

		for i := range rawValues {
			valuePtrs[i] = &rawValues[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// Convert each value to string
		row := make([]string, len(columns))
		for i, val := range rawValues {
			if val == nil {
				row[i] = "NULL"
				continue
			}

			switch v := val.(type) {
			case []byte:
				// Most DB drivers return strings as []byte
				row[i] = string(v)
			default:
				row[i] = fmt.Sprintf("%v", v)
			}
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	fmt.Println(len(results))
	return results, nil
}
