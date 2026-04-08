package store

import (
	"context"
	"database/sql"
)

type TipQRCode struct {
	DID       string `json:"did"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	Data      []byte `json:"-"`
	UpdatedAt string `json:"updated_at"`
}

func (s *Store) UpsertTipQRCode(ctx context.Context, tip TipQRCode) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO tip_qrcodes(did, filename, mime_type, data, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(did) DO UPDATE SET
			filename = excluded.filename,
			mime_type = excluded.mime_type,
			data = excluded.data,
			updated_at = excluded.updated_at`,
		tip.DID,
		tip.Filename,
		tip.MimeType,
		tip.Data,
		tip.UpdatedAt,
	)
	return err
}

func (s *Store) GetTipQRCode(ctx context.Context, did string) (TipQRCode, error) {
	var tip TipQRCode
	err := s.db.QueryRowContext(
		ctx,
		`SELECT did, filename, mime_type, data, updated_at
		 FROM tip_qrcodes
		 WHERE did = ?`,
		did,
	).Scan(&tip.DID, &tip.Filename, &tip.MimeType, &tip.Data, &tip.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return TipQRCode{}, ErrNotFound
		}
		return TipQRCode{}, err
	}
	return tip, nil
}

func (s *Store) DeleteTipQRCode(ctx context.Context, did string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tip_qrcodes WHERE did = ?`, did)
	return err
}

func (s *Store) HasTipQRCode(ctx context.Context, did string) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM tip_qrcodes WHERE did = ?`, did).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}
