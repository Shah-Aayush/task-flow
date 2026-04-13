// Package migrations provides the embedded SQL migration and seed files.
// This package lives at internal/migrations so main.go can import it without
// needing //go:embed directives that traverse up with ".." — which the Go
// embed spec does not permit.
//
// The actual SQL files are located at:
//   - backend/migrations/*.sql  (referenced as ../../migrations/*.sql from this file)
//   - backend/seeds/seed.sql    (referenced as ../../seeds/seed.sql from this file)
//
// Note: Go's embed package DOES allow ".." in paths as long as they remain
// within the module root directory. These paths are valid because the module
// root is backend/ which contains both internal/ and migrations/.
package migrations

import "embed"

// FS contains all migration SQL files, embedded at build time.
// Using embed.FS bakes the migration files into the binary so the server can
// run migrations without external file access — enabling the distroless image.
//
//go:embed sql/migrations/*.sql
var FS embed.FS

// SeedSQL contains the seed data SQL, embedded at build time.
//
//go:embed sql/seed.sql
var SeedSQL string
