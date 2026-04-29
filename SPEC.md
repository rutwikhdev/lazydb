fzf like Database browser tui

### Architecture
- Result Engine
- drivers package
- saved queries/query builder
- responsive tui using bubbletea

### Features:
1. Search tables
2. open tables to view entries(clicking enter)
3. operations
    - search records in table view
    - update records
4. async fetch for long tables(fetch 20 as we scroll down)

Support:
- Mysql
- Mariadb
- Postgres
- Sqlite
 
Stack:
- Golang
    - koanf for config management
    - bubbletea for tui

- Use [[indexer.md]]

Keymaps:
- Ctrl + T to fuzzy search tables. Enter opens the table view
- "/" searches records in the table
