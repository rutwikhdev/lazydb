package db

import "testing"

func TestBuildDSNSQLite(t *testing.T) {
	got, err := BuildDSN(ConnectionInfo{Type: SQLITE, Path: "/tmp/test.db"})
	if err != nil {
		t.Fatalf("BuildDSN returned error: %v", err)
	}
	if got != "/tmp/test.db" {
		t.Fatalf("BuildDSN sqlite = %q, want %q", got, "/tmp/test.db")
	}
}

func TestBuildDSNMySQL(t *testing.T) {
	got, err := BuildDSN(ConnectionInfo{
		Type:     MYSQL,
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "secret",
		Database: "lazydb",
	})
	if err != nil {
		t.Fatalf("BuildDSN returned error: %v", err)
	}
	if got != "root:secret@tcp(localhost:3306)/lazydb" {
		t.Fatalf("BuildDSN mysql = %q", got)
	}
}

func TestBuildDSNPostgres(t *testing.T) {
	got, err := BuildDSN(ConnectionInfo{
		Type:     POSTGRES,
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Password: "secret",
		Database: "lazydb",
	})
	if err != nil {
		t.Fatalf("BuildDSN returned error: %v", err)
	}
	want := "host=localhost port=5432 user=postgres password=secret dbname=lazydb sslmode=disable"
	if got != want {
		t.Fatalf("BuildDSN postgres = %q, want %q", got, want)
	}
}

func TestBuildDSNPostgresDefaultsDatabase(t *testing.T) {
	got, err := BuildDSN(ConnectionInfo{
		Type:     POSTGRES,
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("BuildDSN returned error: %v", err)
	}
	want := "host=localhost port=5432 user=postgres password=secret dbname=postgres sslmode=disable"
	if got != want {
		t.Fatalf("BuildDSN postgres default database = %q, want %q", got, want)
	}
}

func TestBuildDSNUnsupportedType(t *testing.T) {
	if got, err := BuildDSN(ConnectionInfo{Type: "oracle"}); err == nil || got != "" {
		t.Fatalf("expected unsupported database error and empty dsn, got dsn=%q err=%v", got, err)
	}
}
