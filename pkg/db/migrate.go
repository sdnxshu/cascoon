// pkg/db/migrate.go
package db

import "fmt"

func Migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS runs (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			repo        TEXT NOT NULL,
			status      TEXT NOT NULL DEFAULT 'pending',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			finished_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS run_logs (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			run_id      UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
			workflow    TEXT NOT NULL,
			step        TEXT NOT NULL,
			output      TEXT,
			exit_code   INT NOT NULL DEFAULT 0,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_run_logs_run_id ON run_logs(run_id)`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	return nil
}
