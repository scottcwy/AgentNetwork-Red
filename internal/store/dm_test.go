package store

import (
	"context"
	"testing"
	"time"
)

func TestInsertDirectMessageDedupAndQuery(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	msg := DirectMessage{
		ID:             "msg-1",
		FromDID:        "did:key:sender",
		ToDID:          "did:key:receiver",
		Ciphertext:     "cipher",
		Nonce:          "nonce",
		Algo:           "nacl",
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		LocalPlaintext: "hello",
	}

	if err := st.InsertDirectMessage(ctx, msg); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertDirectMessage(ctx, msg); err != nil {
		t.Fatal(err)
	}

	inbox, err := st.Inbox(ctx, "did:key:receiver", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(inbox) != 1 {
		t.Fatalf("expected 1 inbox message, got %d", len(inbox))
	}

	thread, err := st.Thread(ctx, "did:key:receiver", "did:key:sender", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(thread) != 1 {
		t.Fatalf("expected 1 thread message, got %d", len(thread))
	}
	if thread[0].LocalPlaintext != "hello" {
		t.Fatalf("unexpected plaintext: %q", thread[0].LocalPlaintext)
	}
}
