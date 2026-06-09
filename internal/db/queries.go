package db

import "fmt"

const (
	QConnString          = "CONN_STRING"
	QFetchTables         = "FETCH_TABLES"
	QFetchDatabases      = "FETCH_DATABASES"
	QFetchPrimaryKey     = "FETCH_PRIMARY_KEY"
	QGetColumns          = "GET_COLUMNS"
	QSelectRows          = "SELECT_ROWS"
	QSelectRowsPaginated = "SELECT_ROWS_PAGINATED"
	QSelectRowByPK       = "SELECT_ROW_BY_PK"
	QDelete              = "DELETE"
	QInsert              = "INSERT"
	QUpdate              = "UPDATE"
	QFetchAutoIncrement  = "FETCH_AUTO_INCREMENT"
)

var QueryMap = map[string]map[string]string{
	SQLITE: {
		QConnString:          "",
		QFetchTables:         "SELECT name FROM sqlite_master WHERE type='table';",
		QFetchDatabases:      "",
		QFetchPrimaryKey:     "SELECT name FROM pragma_table_info('%s') WHERE pk > 0;",
		QGetColumns:          "PRAGMA table_info(%s);",
		QSelectRows:          `SELECT %s FROM %s`,
		QSelectRowsPaginated: `SELECT %s FROM %s LIMIT %d OFFSET %d`,
		QSelectRowByPK:       `SELECT %s FROM %s WHERE %s = %s`,
		QDelete:              `DELETE FROM %s WHERE %s = %s`,
		QInsert:              `INSERT INTO %s (%s) VALUES (%s)`,
		QUpdate:              `UPDATE %s SET %s WHERE %s = %s`,
		QFetchAutoIncrement:  "SELECT name FROM pragma_table_info('%s') WHERE pk > 0 AND UPPER(type) = 'INTEGER';",
	},
	MYSQL: {
		QConnString:          "%s:%s@tcp(%s:%d)/%s",
		QFetchTables:         "SHOW TABLES;",
		QFetchDatabases:      "SHOW DATABASES;",
		QFetchPrimaryKey:     "SELECT column_name FROM information_schema.key_column_usage WHERE table_schema = DATABASE() AND table_name = '%s' AND constraint_name = 'PRIMARY';",
		QGetColumns:          "SHOW COLUMNS FROM `%s`;",
		QSelectRows:          `SELECT %s FROM %s`,
		QSelectRowsPaginated: `SELECT %s FROM %s LIMIT %d OFFSET %d`,
		QSelectRowByPK:       `SELECT %s FROM %s WHERE %s = %s`,
		QDelete:              `DELETE FROM %s WHERE %s = %s`,
		QInsert:              `INSERT INTO %s (%s) VALUES (%s)`,
		QUpdate:              `UPDATE %s SET %s WHERE %s = %s`,
		QFetchAutoIncrement:  "SELECT column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = '%s' AND extra LIKE '%%auto_increment%%';",
	},
	POSTGRES: {
		QConnString:          "host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		QFetchTables:         "SELECT tablename FROM pg_tables WHERE schemaname = 'public';",
		QFetchDatabases:      "SELECT datname FROM pg_database WHERE datistemplate = false;",
		QFetchPrimaryKey:     "SELECT a.attname FROM pg_index i JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey) WHERE i.indrelid = '%s'::regclass AND i.indisprimary;",
		QGetColumns:          "SELECT column_name FROM information_schema.columns WHERE table_name = '%s' AND table_schema = 'public' ORDER BY ordinal_position;",
		QSelectRows:          `SELECT %s FROM %s`,
		QSelectRowsPaginated: `SELECT %s FROM %s LIMIT %d OFFSET %d`,
		QSelectRowByPK:       `SELECT %s FROM %s WHERE %s = %s`,
		QDelete:              `DELETE FROM %s WHERE %s = %s`,
		QInsert:              `INSERT INTO %s (%s) VALUES (%s)`,
		QUpdate:              `UPDATE %s SET %s WHERE %s = %s`,
		QFetchAutoIncrement:  "SELECT column_name FROM information_schema.columns WHERE table_name = '%s' AND table_schema = 'public' AND (column_default LIKE 'nextval(%%' OR is_identity = 'YES');",
	},
}

var IdentifierQuote = map[string]string{
	SQLITE:   `"`,
	MYSQL:    "`",
	POSTGRES: `"`,
}

func GetQuery(dbType, queryKey string) (string, bool) {
	queries, ok := QueryMap[dbType]
	if !ok {
		return "", false
	}
	query, ok := queries[queryKey]
	return query, ok
}

func Placeholder(dbType string, index int) string {
	if dbType == POSTGRES {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func quoteIdent(dbType, name string) string {
	q := IdentifierQuote[dbType]
	return q + name + q
}
