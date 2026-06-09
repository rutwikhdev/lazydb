package db

const (
	SQLITE   = "sqlite"
	MYSQL    = "mysql"
	POSTGRES = "postgres"

	DEFAULT_HOST = "localhost"

	MYSQL_DEFAULT_PORT    = 3306
	POSTGRES_DEFAULT_PORT = 5432

	POSTGRES_INITIAL_DB = "postgres"
)

var DBDisplayNames = map[string]string{
	SQLITE:   "SQLite",
	MYSQL:    "MySQL",
	POSTGRES: "PostgreSQL",
}

var SupportedDBs = []string{
	SQLITE,
	MYSQL,
	POSTGRES,
}
