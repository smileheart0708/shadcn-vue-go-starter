// Package migrations embeds the SQLite migration set.
//
// The files are applied in version order by internal/database/migrate, which
// records the SHA-256 of every applied file: a published file is immutable.
package migrations

import "embed"

// FS holds the <version>_<name>.sql files of the SQLite dialect.
//
//go:embed *.sql
var FS embed.FS
