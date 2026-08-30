package migrations

import "embed"

// FS contains the versioned Goose SQL migrations used by the server and tests.
//
//go:embed *.sql
var FS embed.FS
