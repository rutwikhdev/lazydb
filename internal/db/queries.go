package db

import "fmt"

var FetchTablesQuery = map[string]string{
	DB_SQLITE:   "SELECT name FROM sqlite_master WHERE type='table';",
	DB_MYSQL:    "SHOW TABLES;",
	DB_POSTGRES: "SELECT tablename FROM pg_tables WHERE schemaname = 'public';",
}

var FetchDatabasesQuery = map[string]string{
	DB_SQLITE:   "",
	DB_MYSQL:    "SHOW DATABASES;",
	DB_POSTGRES: "SELECT datname FROM pg_database WHERE datistemplate = false;",
}

var FetchPrimaryKeyQuery = map[string]string{
	DB_SQLITE:   "SELECT name FROM pragma_table_info('%s') WHERE pk > 0;",
	DB_MYSQL:    "SELECT column_name FROM information_schema.key_column_usage WHERE table_schema = DATABASE() AND table_name = '%s' AND constraint_name = 'PRIMARY';",
	DB_POSTGRES: "SELECT a.attname FROM pg_index i JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey) WHERE i.indrelid = '%s'::regclass AND i.indisprimary;",
}

var IdentifierQuote = map[string]string{
	DB_SQLITE:   `"`,
	DB_MYSQL:    "`",
	DB_POSTGRES: `"`,
}

func Placeholder(dbType string, index int) string {
	if dbType == DB_POSTGRES {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func quoteIdent(dbType, name string) string {
	q := IdentifierQuote[dbType]
	return q + name + q
}

var DeleteQueryTemplate = map[string]string{
	DB_SQLITE:   `DELETE FROM %s WHERE %s = %s`,
	DB_MYSQL:    `DELETE FROM %s WHERE %s = %s`,
	DB_POSTGRES: `DELETE FROM %s WHERE %s = %s`,
}

var UpdateQueryTemplate = map[string]string{
	DB_SQLITE:   `UPDATE %s SET %s WHERE %s = %s`,
	DB_MYSQL:    `UPDATE %s SET %s WHERE %s = %s`,
	DB_POSTGRES: `UPDATE %s SET %s WHERE %s = %s`,
}

var SelectRowByPKQuery = map[string]string{
	DB_SQLITE:   `SELECT %s FROM %s WHERE %s = %s`,
	DB_MYSQL:    `SELECT %s FROM %s WHERE %s = %s`,
	DB_POSTGRES: `SELECT %s FROM %s WHERE %s = %s`,
}

var SelectRowsQuery = map[string]string{
	DB_SQLITE:   `SELECT %s FROM %s`,
	DB_MYSQL:    `SELECT %s FROM %s`,
	DB_POSTGRES: `SELECT %s FROM %s`,
}

var SelectRowsPaginatedQuery = map[string]string{
	DB_SQLITE:   `SELECT %s FROM %s LIMIT %d OFFSET %d`,
	DB_MYSQL:    `SELECT %s FROM %s LIMIT %d OFFSET %d`,
	DB_POSTGRES: `SELECT %s FROM %s LIMIT %d OFFSET %d`,
}
