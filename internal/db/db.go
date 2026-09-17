package db

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/lib/pq"

	"debate/internal/orchestrator"
)

type Store struct {
	conn *sql.DB
}

func Open(dsn string) (*Store, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	s := &Store{conn: conn}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.conn.Exec(`
		CREATE TABLE IF NOT EXISTS debates (
			id           SERIAL PRIMARY KEY,
			question     TEXT NOT NULL,
			debate_json  JSONB NOT NULL,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	return err
}

func (s *Store) SaveDebate(d *orchestrator.Debate) (int64, error) {
	body, err := json.Marshal(d)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.conn.QueryRow(
		`INSERT INTO debates (question, debate_json) VALUES ($1, $2) RETURNING id`,
		d.Question, body,
	).Scan(&id)
	return id, err
}

func (s *Store) GetDebate(id int64) (*orchestrator.Debate, error) {
	var body []byte
	err := s.conn.QueryRow(`SELECT debate_json FROM debates WHERE id = $1`, id).Scan(&body)
	if err != nil {
		return nil, err
	}
	var d orchestrator.Debate
	if err := json.Unmarshal(body, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

type DebateSummary struct {
	ID       int64  `json:"id"`
	Question string `json:"question"`
}

func (s *Store) ListDebates(limit int) ([]DebateSummary, error) {
	rows, err := s.conn.Query(
		`SELECT id, question FROM debates ORDER BY created_at DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DebateSummary
	for rows.Next() {
		var d DebateSummary
		if err := rows.Scan(&d.ID, &d.Question); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
