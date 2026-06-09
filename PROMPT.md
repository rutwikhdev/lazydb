I want to refactor all database related queries from internal/db/db.go to internal/db/queries.go

queries.go should build a structure for queries, so that we can fetch the relevant query. I prefer a map like,
```json
QueryMap = {
  "POSTGRES": {
    "CONN_STRING": "pg_conn_string",
    "GET_COLUMNS": "get_columns_query"
  },
  "SQLITE": {
    "CONN_STRING": "sqlite_conn_string",
    "GET_COLUMNS": "sqlite_columns_query"
  },
  "MYSQL": {}
}
```
If you have any other ideas let me know

Note: Refactor all the loose db queries and conn strings from db.go to queries.go

Plan this
