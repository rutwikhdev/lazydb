package db

const (
	DB_SQLITE   = "sqlite"
	DB_MYSQL    = "mysql"
	DB_MARIADB  = "mariadb"
	DB_POSTGRES = "postgres"

	DEFAULT_HOST = "localhost"

	MYSQL_DEFAULT_PORT    = 3306
	MARIADB_DEFAULT_PORT  = 3306
	POSTGRES_DEFAULT_PORT = 5432

	POSTGRES_INITIAL_DB = "postgres"
)

var DBDisplayNames = map[string]string{
	DB_SQLITE:   "SQLite",
	DB_MYSQL:    "MySQL",
	DB_MARIADB:  "MariaDB",
	DB_POSTGRES: "PostgreSQL",
}

var SupportedDBs = []string{
	DB_SQLITE,
	DB_MYSQL,
	DB_MARIADB,
	DB_POSTGRES,
}
