// Package migrations provides database migration utilities and embedded migration files.
//
// FS is an embedded filesystem containing migration files used by the application.
package migrations

import "embed"

//go:embed *
var FS embed.FS
