package lib

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Session struct {
	ID            string
	Workflow      string
	Target        string
	Metric        string
	Objective     string
	Prompt        string
	MaxIterations int
	BudgetUSD     float64
	Status        string // pending|running|paused|done|failed
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Step struct {
	ID             int
	SessionID      string
	Iteration      int
	Status         string // running|kept|reverted|failed
	AgentName      string
	MetricOutput   string
	MetricExitCode int
	Kept           bool
	TokensUsed     int
	CostUSD        float64
	ChangeSummary  string
	StartedAt      time.Time
	FinishedAt     time.Time
}

type DB struct {
	conn *sql.DB
	path string
}

func OpenDB(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	conn, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db := &DB{conn: conn, path: path}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}
	return db, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id            TEXT PRIMARY KEY,
		workflow      TEXT NOT NULL,
		target        TEXT NOT NULL DEFAULT '',
		metric        TEXT NOT NULL DEFAULT '',
		objective     TEXT NOT NULL DEFAULT '',
		prompt        TEXT NOT NULL DEFAULT '',
		max_iterations INTEGER NOT NULL DEFAULT 0,
		budget_usd    REAL NOT NULL DEFAULT 0,
		status        TEXT NOT NULL DEFAULT 'pending',
		created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS steps (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id      TEXT NOT NULL REFERENCES sessions(id),
		iteration       INTEGER NOT NULL,
		status          TEXT NOT NULL DEFAULT 'running',
		agent_name      TEXT NOT NULL DEFAULT 'claude',
		metric_output   TEXT NOT NULL DEFAULT '',
		metric_exit_code INTEGER NOT NULL DEFAULT -1,
		kept            BOOLEAN NOT NULL DEFAULT 0,
		tokens_used     INTEGER NOT NULL DEFAULT 0,
		cost_usd        REAL NOT NULL DEFAULT 0,
		change_summary  TEXT NOT NULL DEFAULT '',
		started_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		finished_at     DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_steps_session ON steps(session_id);
	`
	_, err := d.conn.Exec(schema)
	return err
}

func (d *DB) CreateSession(s *Session) error {
	s.CreatedAt = time.Now()
	s.UpdatedAt = s.CreatedAt
	_, err := d.conn.Exec(
		`INSERT INTO sessions (id, workflow, target, metric, objective, prompt, max_iterations, budget_usd, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Workflow, s.Target, s.Metric, s.Objective, s.Prompt, s.MaxIterations, s.BudgetUSD, s.Status, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (d *DB) UpdateSessionStatus(id, status string) error {
	_, err := d.conn.Exec(
		`UPDATE sessions SET status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now(), id,
	)
	return err
}

func (d *DB) GetSession(id string) (*Session, error) {
	s := &Session{}
	err := d.conn.QueryRow(
		`SELECT id, workflow, target, metric, objective, prompt, max_iterations, budget_usd, status, created_at, updated_at FROM sessions WHERE id = ?`, id,
	).Scan(&s.ID, &s.Workflow, &s.Target, &s.Metric, &s.Objective, &s.Prompt, &s.MaxIterations, &s.BudgetUSD, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session %s not found", id)
	}
	return s, err
}

func (d *DB) ListSessions() ([]Session, error) {
	rows, err := d.conn.Query(
		`SELECT id, workflow, target, metric, objective, prompt, max_iterations, budget_usd, status, created_at, updated_at
		 FROM sessions ORDER BY created_at DESC LIMIT 50`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.Workflow, &s.Target, &s.Metric, &s.Objective, &s.Prompt, &s.MaxIterations, &s.BudgetUSD, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (d *DB) CreateStep(s *Step) error {
	s.StartedAt = time.Now()
	res, err := d.conn.Exec(
		`INSERT INTO steps (session_id, iteration, status, agent_name, started_at) VALUES (?, ?, ?, ?, ?)`,
		s.SessionID, s.Iteration, s.Status, s.AgentName, s.StartedAt,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	s.ID = int(id)
	return nil
}

func (d *DB) UpdateStep(s *Step) error {
	s.FinishedAt = time.Now()
	_, err := d.conn.Exec(
		`UPDATE steps SET status = ?, metric_output = ?, metric_exit_code = ?, kept = ?, tokens_used = ?, cost_usd = ?, change_summary = ?, finished_at = ?
		 WHERE id = ?`,
		s.Status, s.MetricOutput, s.MetricExitCode, s.Kept, s.TokensUsed, s.CostUSD, s.ChangeSummary, s.FinishedAt, s.ID,
	)
	return err
}

func (d *DB) GetSteps(sessionID string) ([]Step, error) {
	rows, err := d.conn.Query(
		`SELECT id, session_id, iteration, status, agent_name, metric_output, metric_exit_code, kept, tokens_used, cost_usd, change_summary, started_at, COALESCE(finished_at, '') as finished_at
		 FROM steps WHERE session_id = ? ORDER BY iteration`, sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []Step
	for rows.Next() {
		var s Step
		var finishedStr string
		if err := rows.Scan(&s.ID, &s.SessionID, &s.Iteration, &s.Status, &s.AgentName, &s.MetricOutput, &s.MetricExitCode, &s.Kept, &s.TokensUsed, &s.CostUSD, &s.ChangeSummary, &s.StartedAt, &finishedStr); err != nil {
			return nil, err
		}
		if finishedStr != "" {
			s.FinishedAt, _ = time.Parse(time.RFC3339, finishedStr)
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

func (d *DB) SessionTotalCost(sessionID string) (float64, error) {
	var total float64
	err := d.conn.QueryRow(
		`SELECT COALESCE(SUM(cost_usd), 0) FROM steps WHERE session_id = ?`, sessionID,
	).Scan(&total)
	return total, err
}

func (d *DB) LastIteration(sessionID string) (int, error) {
	var iter int
	err := d.conn.QueryRow(
		`SELECT COALESCE(MAX(iteration), 0) FROM steps WHERE session_id = ?`, sessionID,
	).Scan(&iter)
	return iter, err
}
