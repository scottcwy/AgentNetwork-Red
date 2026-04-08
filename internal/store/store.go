package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const databaseFile = "redanet.db"

const InitialBalance int64 = 10000

type Store struct {
	db   *sql.DB
	path string
}

type CreditEvent struct {
	ID        string `json:"id"`
	PeerDID   string `json:"peer_did"`
	Delta     int64  `json:"delta"`
	Balance   int64  `json:"balance"`
	Reason    string `json:"reason,omitempty"`
	RefID     string `json:"ref_id,omitempty"`
	CreatedAt string `json:"created_at"`
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, databaseFile)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := pingWithTimeout(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	s := &Store{db: db, path: dbPath}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Path() string {
	return s.path
}

func pingWithTimeout(db *sql.DB) error {
	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := db.Ping(); err == nil {
			return nil
		} else if time.Now().After(deadline) {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		return fmt.Errorf("enable wal: %w", err)
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			applied_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			publisher TEXT NOT NULL,
			claimant TEXT,
			reward INTEGER NOT NULL,
			tier TEXT NOT NULL DEFAULT 'micro',
			mode TEXT NOT NULL DEFAULT 'simple',
			state TEXT NOT NULL DEFAULT 'created',
			result TEXT,
			escrow_amount INTEGER DEFAULT 0,
			deposit_amount INTEGER DEFAULT 0,
			tags TEXT,
			created_at TEXT NOT NULL,
			claimed_at TEXT,
			submitted_at TEXT,
			settled_at TEXT,
			state_entered_at TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS credit_events (
			id TEXT PRIMARY KEY,
			peer_did TEXT NOT NULL,
			delta INTEGER NOT NULL,
			balance INTEGER NOT NULL,
			reason TEXT,
			ref_id TEXT,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS direct_messages (
			id TEXT PRIMARY KEY,
			from_did TEXT NOT NULL,
			to_did TEXT NOT NULL,
			ciphertext TEXT NOT NULL,
			nonce TEXT,
			algo TEXT DEFAULT 'nacl',
			read INTEGER DEFAULT 0,
			timestamp TEXT NOT NULL,
			local_plaintext TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS tip_qrcodes (
			did TEXT PRIMARY KEY,
			filename TEXT NOT NULL,
			mime_type TEXT NOT NULL,
			data BLOB NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_state ON tasks(state);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_publisher ON tasks(publisher);`,
		`CREATE INDEX IF NOT EXISTS idx_credits_peer ON credit_events(peer_did);`,
		`CREATE INDEX IF NOT EXISTS idx_dm_to ON direct_messages(to_did, timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_dm_thread ON direct_messages(from_did, to_did);`,
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	if _, err := tx.Exec(
		`INSERT OR IGNORE INTO schema_migrations(name, applied_at) VALUES (?, ?)`,
		"bootstrap",
		time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (s *Store) EnsureInitialBalance(ctx context.Context, did string, amount int64) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM credit_events WHERE peer_did = ?`, did).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO credit_events(id, peer_did, delta, balance, reason, ref_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"genesis:"+did,
		did,
		amount,
		amount,
		"initial_balance",
		"",
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) Balance(ctx context.Context, did string) (int64, error) {
	var balance int64
	err := s.db.QueryRowContext(
		ctx,
		`SELECT balance
		 FROM credit_events
		 WHERE peer_did = ?
		 ORDER BY rowid DESC
		 LIMIT 1`,
		did,
	).Scan(&balance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return balance, nil
}

func (s *Store) ListCreditEvents(ctx context.Context, did string, limit int) ([]CreditEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, peer_did, delta, balance, reason, ref_id, created_at
		 FROM credit_events
		 WHERE peer_did = ?
		 ORDER BY rowid DESC
		 LIMIT ?`,
		did,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []CreditEvent
	for rows.Next() {
		var event CreditEvent
		if err := rows.Scan(
			&event.ID,
			&event.PeerDID,
			&event.Delta,
			&event.Balance,
			&event.Reason,
			&event.RefID,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}
