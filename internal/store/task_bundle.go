package store

import (
	"context"
	"database/sql"
	"errors"
)

type TaskBundle struct {
	TaskID     string `json:"task_id"`
	Filename   string `json:"filename"`
	MimeType   string `json:"mime_type"`
	Data       []byte `json:"-"`
	Uploader   string `json:"uploader"`
	UploadedAt string `json:"uploaded_at"`
}

func (s *Store) UpsertTaskBundle(ctx context.Context, bundle TaskBundle) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO task_bundles(task_id, filename, mime_type, data, uploader, uploaded_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(task_id) DO UPDATE SET
		   filename=excluded.filename,
		   mime_type=excluded.mime_type,
		   data=excluded.data,
		   uploader=excluded.uploader,
		   uploaded_at=excluded.uploaded_at`,
		bundle.TaskID,
		bundle.Filename,
		bundle.MimeType,
		bundle.Data,
		bundle.Uploader,
		bundle.UploadedAt,
	)
	return err
}

func (s *Store) GetTaskBundle(ctx context.Context, taskID string) (TaskBundle, error) {
	var bundle TaskBundle
	err := s.db.QueryRowContext(
		ctx,
		`SELECT task_id, filename, mime_type, data, uploader, uploaded_at
		 FROM task_bundles
		 WHERE task_id = ?`,
		taskID,
	).Scan(
		&bundle.TaskID,
		&bundle.Filename,
		&bundle.MimeType,
		&bundle.Data,
		&bundle.Uploader,
		&bundle.UploadedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TaskBundle{}, ErrNotFound
		}
		return TaskBundle{}, err
	}
	return bundle, nil
}
