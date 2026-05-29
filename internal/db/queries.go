package db

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
