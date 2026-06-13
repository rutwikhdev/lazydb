package db

import "testing"

func TestGetQueryReturnsKnownQueries(t *testing.T) {
	query, ok := GetQuery(SQLITE, QFetchTables)
	if !ok {
		t.Fatalf("expected sqlite fetch tables query to exist")
	}
	if query == "" {
		t.Fatalf("expected sqlite fetch tables query to be non-empty")
	}
}

func TestGetQueryUnknownDatabase(t *testing.T) {
	if query, ok := GetQuery("unknown", QFetchTables); ok || query != "" {
		t.Fatalf("expected unknown database query lookup to fail, got query=%q ok=%v", query, ok)
	}
}

func TestGetQueryUnknownKey(t *testing.T) {
	if query, ok := GetQuery(SQLITE, "UNKNOWN"); ok || query != "" {
		t.Fatalf("expected unknown query key lookup to fail, got query=%q ok=%v", query, ok)
	}
}

func TestPlaceholder(t *testing.T) {
	tests := []struct {
		name   string
		dbType string
		index  int
		want   string
	}{
		{name: "sqlite", dbType: SQLITE, index: 1, want: "?"},
		{name: "mysql", dbType: MYSQL, index: 2, want: "?"},
		{name: "postgres first", dbType: POSTGRES, index: 1, want: "$1"},
		{name: "postgres second", dbType: POSTGRES, index: 2, want: "$2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Placeholder(tt.dbType, tt.index); got != tt.want {
				t.Fatalf("Placeholder(%q, %d) = %q, want %q", tt.dbType, tt.index, got, tt.want)
			}
		})
	}
}

func TestQuoteIdent(t *testing.T) {
	tests := []struct {
		name   string
		dbType string
		ident  string
		want   string
	}{
		{name: "sqlite", dbType: SQLITE, ident: "users", want: `"users"`},
		{name: "mysql", dbType: MYSQL, ident: "users", want: "`users`"},
		{name: "postgres", dbType: POSTGRES, ident: "users", want: `"users"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quoteIdent(tt.dbType, tt.ident); got != tt.want {
				t.Fatalf("quoteIdent(%q, %q) = %q, want %q", tt.dbType, tt.ident, got, tt.want)
			}
		})
	}
}

func TestQueryMapHasRequiredKeys(t *testing.T) {
	required := []string{
		QConnString,
		QFetchTables,
		QFetchDatabases,
		QFetchPrimaryKey,
		QGetColumns,
		QSelectRows,
		QSelectRowsPaginated,
		QSelectRowByPK,
		QDelete,
		QInsert,
		QUpdate,
		QFetchAutoIncrement,
	}

	for _, dbType := range SupportedDBs {
		t.Run(dbType, func(t *testing.T) {
			queries, ok := QueryMap[dbType]
			if !ok {
				t.Fatalf("missing query map for %s", dbType)
			}
			for _, key := range required {
				if _, ok := queries[key]; !ok {
					t.Fatalf("missing query %s for %s", key, dbType)
				}
			}
		})
	}
}
