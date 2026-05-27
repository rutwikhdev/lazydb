package db

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type ConnectionInfo struct {
	Type     string
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Path     string
}

type Database struct {
	DB     *sql.DB
	Type   string
	Conn   ConnectionInfo
}

func BuildDSN(info ConnectionInfo) (string, error) {
	switch info.Type {
	case DB_SQLITE:
		return info.Path, nil
	case DB_MYSQL:
		dbName := info.Database
		if dbName == "" {
			dbName = ""
		}
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			info.Username, info.Password, info.Host, info.Port, dbName), nil
	case DB_POSTGRES:
		dbName := info.Database
		if dbName == "" {
			dbName = POSTGRES_INITIAL_DB
		}
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			info.Host, info.Port, info.Username, info.Password, dbName), nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", info.Type)
	}
}

func NewDBFromInfo(info ConnectionInfo) (*Database, error) {
	dsn, err := BuildDSN(info)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(info.Type, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{DB: db, Type: info.Type, Conn: info}, nil
}

func (db *Database) FetchTables() []string {
	query, ok := Tables[db.Type]
	if !ok || len(query) == 0 {
		log.Fatalf("database type %s not supported for listing tables", db.Type)
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

func (db *Database) FetchDatabases() ([]string, error) {
	query, ok := Databases[db.Type]
	if !ok || len(query) == 0 {
		return nil, fmt.Errorf("database type %s does not support listing databases", db.Type)
	}

	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	databases := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		databases = append(databases, name)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return databases, nil
}

func (db *Database) SelectDatabase(name string) error {
	switch db.Type {
	case DB_MYSQL:
		_, err := db.DB.Exec("USE " + name)
		if err != nil {
			return err
		}
		db.Conn.Database = name
		return nil
	case DB_POSTGRES:
		db.Conn.Database = name
		dsn, err := BuildDSN(db.Conn)
		if err != nil {
			return err
		}
		newDB, err := sql.Open(db.Type, dsn)
		if err != nil {
			return fmt.Errorf("failed to open new database: %w", err)
		}
		if err := newDB.Ping(); err != nil {
			newDB.Close()
			return fmt.Errorf("failed to ping new database: %w", err)
		}
		db.DB.Close()
		db.DB = newDB
		return nil
	case DB_SQLITE:
		// SQLite is file-based; no-op
		return nil
	default:
		return fmt.Errorf("unsupported database type: %s", db.Type)
	}
}

func (db *Database) GetColumns(table string) []string {
	var query string
	switch db.Type {
	case DB_SQLITE:
		query = fmt.Sprintf("PRAGMA table_info(%s);", table)
	case DB_MYSQL:
		query = fmt.Sprintf("SHOW COLUMNS FROM %s;", table)
	case DB_POSTGRES:
		query = fmt.Sprintf("SELECT column_name FROM information_schema.columns WHERE table_name = '%s';", table)
	default:
		log.Fatalf("unsupported database type for columns: %s", db.Type)
	}

	rows, err := db.DB.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var columns []string

	for rows.Next() {
		var name string
		switch db.Type {
		case DB_SQLITE:
			var cid, notnull, pk int
			var ctype string
			var dfltValue any
			err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
			if err != nil {
				log.Fatal(err)
			}
		case DB_MYSQL:
			var fieldType, null, key, extra string
			var defaultValue any
			err := rows.Scan(&name, &fieldType, &null, &key, &defaultValue, &extra)
			if err != nil {
				log.Fatal(err)
			}
		case DB_POSTGRES:
			err := rows.Scan(&name)
			if err != nil {
				log.Fatal(err)
			}
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

	return scanRows(rows, columns)
}

func (db *Database) GetRowsPaginated(table string, columns []string, offset, limit int) ([][]string, error) {
	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns provided")
	}

	query := fmt.Sprintf("SELECT %s FROM %s LIMIT %d OFFSET %d",
		strings.Join(columns, ", "),
		table,
		limit,
		offset,
	)

	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRows(rows, columns)
}

func scanRows(rows *sql.Rows, columns []string) ([][]string, error) {
	var results [][]string

	for rows.Next() {
		rawValues := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))

		for i := range rawValues {
			valuePtrs[i] = &rawValues[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make([]string, len(columns))
		for i, val := range rawValues {
			if val == nil {
				row[i] = "NULL"
				continue
			}

			switch v := val.(type) {
			case []byte:
				row[i] = string(v)
			case int64:
				row[i] = strconv.FormatInt(v, 10)
			case float64:
				row[i] = strconv.FormatFloat(v, 'f', -1, 64)
			default:
				row[i] = fmt.Sprintf("%v", v)
			}
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
