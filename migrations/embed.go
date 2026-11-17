// Package migrations embeds the SQL files used to build the database schema.
package migrations

import "embed"

// FS contains every migration in this directory.
//
//go:embed *.sql
var FS embed.FS
