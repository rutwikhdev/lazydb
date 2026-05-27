package db

var Tables = map[string]string{
	DB_SQLITE:   "SELECT name FROM sqlite_master WHERE type='table';",
	DB_MYSQL:    "SHOW TABLES;",
	DB_POSTGRES: "SELECT tablename FROM pg_tables WHERE schemaname = 'public';",
}

var Databases = map[string]string{
	DB_SQLITE:   "",
	DB_MYSQL:    "SHOW DATABASES;",
	DB_POSTGRES: "SELECT datname FROM pg_database WHERE datistemplate = false;",
}
