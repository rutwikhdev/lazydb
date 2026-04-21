package db

var Tables = map[string]string{
	"sqlite":   "SELECT name FROM sqlite_master WHERE type='table';",
	"mysql":    "",
	"postgres": "",
	"mariadb":  "",
	"duckdb":   "",
}
