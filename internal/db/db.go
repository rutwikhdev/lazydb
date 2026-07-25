package db

import (
	"database/sql"
	"fmt"
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
	DB   *sql.DB
	Type string
	Conn ConnectionInfo
}

func BuildDSN(info ConnectionInfo) (string, error) {
	switch info.Type {
	case SQLITE:
		return info.Path, nil
	case MYSQL:
		template, ok := GetQuery(MYSQL, QConnString)
		if !ok {
			return "", fmt.Errorf("connection string template not found for mysql")
		}
		return fmt.Sprintf(template,
			info.Username, info.Password, info.Host, info.Port, info.Database), nil
	case POSTGRES:
		dbName := info.Database
		if dbName == "" {
			dbName = POSTGRES_INITIAL_DB
		}
		template, ok := GetQuery(POSTGRES, QConnString)
		if !ok {
			return "", fmt.Errorf("connection string template not found for postgres")
		}
		return fmt.Sprintf(template,
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

func (db *Database) FetchTables() ([]string, error) {
	query, ok := GetQuery(db.Type, QFetchTables)
	if !ok || len(query) == 0 {
		return nil, fmt.Errorf("database type %s not supported for listing tables", db.Type)
	}

	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("could not query database %s", err.Error())
	}
	defer rows.Close()

	tables := []string{}

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table rows %s", err.Error())
		}
		tables = append(tables, tableName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("unable to fetch tables")
	}

	return tables, nil
}

func (db *Database) FetchDatabases() ([]string, error) {
	query, ok := GetQuery(db.Type, QFetchDatabases)
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
	case MYSQL:
		_, err := db.DB.Exec("USE " + name)
		if err != nil {
			return err
		}
		db.Conn.Database = name
		return nil
	case POSTGRES:
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
	case SQLITE:
		return nil
	default:
		return fmt.Errorf("unsupported database type: %s", db.Type)
	}
}

func (db *Database) GetColumns(table string) ([]string, error) {
	query, ok := GetQuery(db.Type, QGetColumns)
	if !ok {
		return nil, fmt.Errorf("unsupported database type for columns: %s", db.Type)
	}
	query = fmt.Sprintf(query, table)

	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("could not execute query %s", err.Error())
	}
	defer rows.Close()

	var columns []string

	for rows.Next() {
		var name string
		switch db.Type {
		case SQLITE:
			var cid, notnull, pk int
			var ctype string
			var dfltValue any
			err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
			if err != nil {
				return nil, fmt.Errorf("could not fetch rows from sqlite, %s", err.Error())
			}
		case MYSQL:
			var fieldType, null, key, extra string
			var defaultValue any
			err := rows.Scan(&name, &fieldType, &null, &key, &defaultValue, &extra)
			if err != nil {
				return nil, fmt.Errorf("could not fetch rows from mysql, %s", err.Error())
			}
		case POSTGRES:
			err := rows.Scan(&name)
			if err != nil {
				return nil, fmt.Errorf("could not fetch rows from postgres, %s", err.Error())
			}
		}
		columns = append(columns, name)
	}

	return columns, nil
}

// quotedColumns returns the columns quoted and joined for use in a SELECT list.
func (db *Database) quotedColumns(columns []string) string {
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = quoteIdent(db.Type, col)
	}
	return strings.Join(quoted, ", ")
}

// queryRows runs a select query and scans the results into string rows.
func (db *Database) queryRows(query string, columns []string, args ...any) ([][]string, error) {
	rows, err := db.DB.Query(query, args...)
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

	template, ok := GetQuery(db.Type, QSelectRowsPaginated)
	if !ok {
		return nil, fmt.Errorf("unsupported database type for select: %s", db.Type)
	}

	query := fmt.Sprintf(template,
		db.quotedColumns(columns),
		quoteIdent(db.Type, table),
		limit,
		offset,
	)

	return db.queryRows(query, columns)
}

func (db *Database) GetRowsMultiFiltered(table string, columns []string, filters map[string]string, offset, limit int) ([][]string, error) {
	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns provided")
	}

	var whereClauses []string
	var args []any
	placeholderIdx := 1
	for _, col := range columns {
		val, ok := filters[col]
		if !ok || val == "" {
			continue
		}
		whereClauses = append(whereClauses,
			fmt.Sprintf("%s %s %s", quoteIdent(db.Type, col), LikeOperator(db.Type), Placeholder(db.Type, placeholderIdx)))
		args = append(args, "%"+val+"%")
		placeholderIdx++
	}

	if len(whereClauses) == 0 {
		return db.GetRowsPaginated(table, columns, offset, limit)
	}

	template, ok := GetQuery(db.Type, QSelectRowsFiltered)
	if !ok {
		return nil, fmt.Errorf("unsupported database type for filtered select: %s", db.Type)
	}

	query := fmt.Sprintf(template,
		db.quotedColumns(columns),
		quoteIdent(db.Type, table),
		strings.Join(whereClauses, " AND "),
		limit,
		offset,
	)

	return db.queryRows(query, columns, args...)
}

func (db *Database) GetPrimaryKey(table string) string {
	query, ok := GetQuery(db.Type, QFetchPrimaryKey)
	if !ok {
		return ""
	}
	query = fmt.Sprintf(query, table)

	var pkCol string
	err := db.DB.QueryRow(query).Scan(&pkCol)
	if err != nil {
		return ""
	}
	return pkCol
}

func (db *Database) GetRowByPK(table string, columns []string, pkCol string, pkVal string) ([]string, error) {
	template, ok := GetQuery(db.Type, QSelectRowByPK)
	if !ok {
		return nil, fmt.Errorf("unsupported database type for select: %s", db.Type)
	}

	query := fmt.Sprintf(template,
		db.quotedColumns(columns),
		quoteIdent(db.Type, table),
		quoteIdent(db.Type, pkCol),
		Placeholder(db.Type, 1),
	)

	result, err := db.queryRows(query, columns, pkVal)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("row not found")
	}
	return result[0], nil
}

func (db *Database) DeleteRow(table, pkCol, pkVal string) error {
	template, ok := GetQuery(db.Type, QDelete)
	if !ok {
		return fmt.Errorf("unsupported database type for delete: %s", db.Type)
	}
	query := fmt.Sprintf(template,
		quoteIdent(db.Type, table),
		quoteIdent(db.Type, pkCol),
		Placeholder(db.Type, 1),
	)
	_, err := db.DB.Exec(query, pkVal)
	return err
}

func (db *Database) UpdateRow(table string, columns []string, values []string, pkCol string, pkVal string) error {
	if len(columns) != len(values) {
		return fmt.Errorf("columns and values length mismatch")
	}

	template, ok := GetQuery(db.Type, QUpdate)
	if !ok {
		return fmt.Errorf("unsupported database type for update: %s", db.Type)
	}

	setClauses := make([]string, len(columns))
	args := make([]any, len(columns)+1)
	for i, col := range columns {
		setClauses[i] = fmt.Sprintf("%s = %s", quoteIdent(db.Type, col), Placeholder(db.Type, i+1))
		args[i] = values[i]
	}
	args[len(columns)] = pkVal

	query := fmt.Sprintf(template,
		quoteIdent(db.Type, table),
		strings.Join(setClauses, ", "),
		quoteIdent(db.Type, pkCol),
		Placeholder(db.Type, len(columns)+1),
	)

	_, err := db.DB.Exec(query, args...)
	return err
}

func (db *Database) GetAutoIncrementColumns(table string) ([]string, error) {
	query, ok := GetQuery(db.Type, QFetchAutoIncrement)
	if !ok {
		return nil, fmt.Errorf("unsupported database type for auto-increment check: %s", db.Type)
	}
	query = fmt.Sprintf(query, table)

	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, nil
}

func (db *Database) InsertRow(table string, columns []string, values []string) error {
	if len(columns) != len(values) {
		return fmt.Errorf("columns and values length mismatch")
	}

	template, ok := GetQuery(db.Type, QInsert)
	if !ok {
		return fmt.Errorf("unsupported database type for insert: %s", db.Type)
	}

	quotedCols := make([]string, len(columns))
	placeholders := make([]string, len(columns))
	args := make([]any, len(columns))
	for i, col := range columns {
		quotedCols[i] = quoteIdent(db.Type, col)
		placeholders[i] = Placeholder(db.Type, i+1)
		args[i] = values[i]
	}

	query := fmt.Sprintf(template,
		quoteIdent(db.Type, table),
		strings.Join(quotedCols, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := db.DB.Exec(query, args...)
	return err
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
