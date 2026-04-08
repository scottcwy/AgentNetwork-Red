package store

import (
	"context"
	"testing"
)

func TestTipQRCodeLifecycle(t *testing.T) {
	t.Parallel()

	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	did := "did:key:tip-owner"

	hasQRCode, err := st.HasTipQRCode(ctx, did)
	if err != nil {
		t.Fatal(err)
	}
	if hasQRCode {
		t.Fatal("expected empty tip status before insert")
	}

	err = st.UpsertTipQRCode(ctx, TipQRCode{
		DID:       did,
		Filename:  "tip.png",
		MimeType:  "image/png",
		Data:      []byte("pngdata"),
		UpdatedAt: nowRFC3339(),
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := st.GetTipQRCode(ctx, did)
	if err != nil {
		t.Fatal(err)
	}
	if got.MimeType != "image/png" || got.Filename != "tip.png" {
		t.Fatalf("unexpected tip metadata: %+v", got)
	}
	if string(got.Data) != "pngdata" {
		t.Fatalf("unexpected tip data: %q", string(got.Data))
	}

	hasQRCode, err = st.HasTipQRCode(ctx, did)
	if err != nil {
		t.Fatal(err)
	}
	if !hasQRCode {
		t.Fatal("expected tip status after insert")
	}

	if err := st.DeleteTipQRCode(ctx, did); err != nil {
		t.Fatal(err)
	}

	hasQRCode, err = st.HasTipQRCode(ctx, did)
	if err != nil {
		t.Fatal(err)
	}
	if hasQRCode {
		t.Fatal("expected empty tip status after delete")
	}

	if _, err := st.GetTipQRCode(ctx, did); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
