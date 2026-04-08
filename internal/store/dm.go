package store

import (
	"context"
	"database/sql"
)

type DirectMessage struct {
	ID             string `json:"id"`
	FromDID        string `json:"from"`
	ToDID          string `json:"to"`
	Ciphertext     string `json:"ciphertext"`
	Nonce          string `json:"nonce"`
	Algo           string `json:"algo"`
	Read           bool   `json:"read"`
	Timestamp      string `json:"timestamp"`
	LocalPlaintext string `json:"plaintext,omitempty"`
}

func (s *Store) InsertDirectMessage(ctx context.Context, msg DirectMessage) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO direct_messages(id, from_did, to_did, ciphertext, nonce, algo, read, timestamp, local_plaintext)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ID,
		msg.FromDID,
		msg.ToDID,
		msg.Ciphertext,
		msg.Nonce,
		msg.Algo,
		boolToInt(msg.Read),
		msg.Timestamp,
		msg.LocalPlaintext,
	)
	return err
}

func (s *Store) Inbox(ctx context.Context, did string, limit int) ([]DirectMessage, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, from_did, to_did, ciphertext, nonce, algo, read, timestamp, local_plaintext
		 FROM direct_messages
		 WHERE to_did = ?
		 ORDER BY timestamp DESC, id DESC
		 LIMIT ?`,
		did,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDirectMessages(rows)
}

func (s *Store) Thread(ctx context.Context, selfDID, peerDID string, limit int) ([]DirectMessage, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, from_did, to_did, ciphertext, nonce, algo, read, timestamp, local_plaintext
		 FROM direct_messages
		 WHERE (from_did = ? AND to_did = ?)
		    OR (from_did = ? AND to_did = ?)
		 ORDER BY timestamp ASC, id ASC
		 LIMIT ?`,
		selfDID,
		peerDID,
		peerDID,
		selfDID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDirectMessages(rows)
}

func scanDirectMessages(rows *sql.Rows) ([]DirectMessage, error) {
	var messages []DirectMessage
	for rows.Next() {
		var msg DirectMessage
		var readInt int
		if err := rows.Scan(
			&msg.ID,
			&msg.FromDID,
			&msg.ToDID,
			&msg.Ciphertext,
			&msg.Nonce,
			&msg.Algo,
			&readInt,
			&msg.Timestamp,
			&msg.LocalPlaintext,
		); err != nil {
			return nil, err
		}
		msg.Read = readInt != 0
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
