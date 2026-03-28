// internal/store/run.go
package store

import (
	"database/sql"
	"time"
)

type Run struct {
	ID         string     `json:"id"`
	Repo       string     `json:"repo"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

type RunLog struct {
	ID        string    `json:"id"`
	RunID     string    `json:"run_id"`
	Workflow  string    `json:"workflow"`
	Step      string    `json:"step"`
	Output    string    `json:"output"`
	ExitCode  int       `json:"exit_code"`
	CreatedAt time.Time `json:"created_at"`
}

type RunStore struct {
	db *sql.DB
}

func NewRunStore(db *sql.DB) *RunStore {
	return &RunStore{db: db}
}

func (s *RunStore) Create(repo string) (*Run, error) {
	run := &Run{}
	err := s.db.QueryRow(
		`INSERT INTO runs (repo, status) VALUES ($1, 'pending')
		 RETURNING id, repo, status, created_at`,
		repo,
	).Scan(&run.ID, &run.Repo, &run.Status, &run.CreatedAt)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (s *RunStore) UpdateStatus(id, status string) error {
	var err error
	if status == "success" || status == "failed" {
		_, err = s.db.Exec(
			`UPDATE runs SET status = $1, finished_at = now() WHERE id = $2`,
			status, id,
		)
	} else {
		_, err = s.db.Exec(
			`UPDATE runs SET status = $1 WHERE id = $2`,
			status, id,
		)
	}
	return err
}

func (s *RunStore) InsertLog(runID, workflow, step, output string, exitCode int) error {
	_, err := s.db.Exec(
		`INSERT INTO run_logs (run_id, workflow, step, output, exit_code)
		 VALUES ($1, $2, $3, $4, $5)`,
		runID, workflow, step, output, exitCode,
	)
	return err
}

func (s *RunStore) GetByID(id string) (*Run, error) {
	run := &Run{}
	err := s.db.QueryRow(
		`SELECT id, repo, status, created_at, finished_at FROM runs WHERE id = $1`,
		id,
	).Scan(&run.ID, &run.Repo, &run.Status, &run.CreatedAt, &run.FinishedAt)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (s *RunStore) List() ([]*Run, error) {
	rows, err := s.db.Query(
		`SELECT id, repo, status, created_at, finished_at FROM runs ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*Run
	for rows.Next() {
		run := &Run{}
		if err := rows.Scan(&run.ID, &run.Repo, &run.Status, &run.CreatedAt, &run.FinishedAt); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (s *RunStore) GetLogs(runID string) ([]*RunLog, error) {
	rows, err := s.db.Query(
		`SELECT id, run_id, workflow, step, output, exit_code, created_at
		 FROM run_logs WHERE run_id = $1 ORDER BY created_at ASC`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*RunLog
	for rows.Next() {
		l := &RunLog{}
		if err := rows.Scan(&l.ID, &l.RunID, &l.Workflow, &l.Step, &l.Output, &l.ExitCode, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}