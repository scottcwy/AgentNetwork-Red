package store

import (
	"context"
	"testing"
	"time"
)

func TestTaskBundleRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	uploadedAt := time.Now().UTC().Format(time.RFC3339)
	if err := st.UpsertTaskBundle(ctx, TaskBundle{
		TaskID:     "task-1",
		Filename:   "deliverable.nut",
		MimeType:   "application/octet-stream",
		Data:       []byte("bundle-bytes"),
		Uploader:   "did:key:claimant",
		UploadedAt: uploadedAt,
	}); err != nil {
		t.Fatalf("upsert bundle: %v", err)
	}

	got, err := st.GetTaskBundle(ctx, "task-1")
	if err != nil {
		t.Fatalf("get bundle: %v", err)
	}

	if got.TaskID != "task-1" {
		t.Fatalf("unexpected task id: %s", got.TaskID)
	}
	if got.Filename != "deliverable.nut" {
		t.Fatalf("unexpected filename: %s", got.Filename)
	}
	if got.MimeType != "application/octet-stream" {
		t.Fatalf("unexpected mime type: %s", got.MimeType)
	}
	if string(got.Data) != "bundle-bytes" {
		t.Fatalf("unexpected data: %q", string(got.Data))
	}
	if got.Uploader != "did:key:claimant" {
		t.Fatalf("unexpected uploader: %s", got.Uploader)
	}
	if got.UploadedAt != uploadedAt {
		t.Fatalf("unexpected uploaded_at: %s", got.UploadedAt)
	}
}
