package store

import (
	"context"
	"testing"
)

func TestProfileAndANSDirectoryQueries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if err := st.TouchProfilePresence(ctx, "did:key:alpha", "12D3KooWAlpha"); err != nil {
		t.Fatalf("touch alpha: %v", err)
	}
	alpha, err := st.UpsertProfile(ctx, AgentProfile{
		DID:         "did:key:alpha",
		Name:        "Alice",
		Description: "Rust systems translator",
		Skills:      []string{"translation", "systems", "rust"},
		Tags:        []string{"review", "bilingual"},
		PeerID:      "12D3KooWAlpha",
	})
	if err != nil {
		t.Fatalf("upsert alpha: %v", err)
	}
	if alpha.Name != "Alice" || len(alpha.Skills) != 3 {
		t.Fatalf("unexpected alpha profile: %+v", alpha)
	}

	record, err := st.RegisterANS(ctx, "alice", "did:key:alpha", []string{"translation", "rust"})
	if err != nil {
		t.Fatalf("register ans: %v", err)
	}
	if record.Name != "agent://alice" || record.DID != "did:key:alpha" {
		t.Fatalf("unexpected ans record: %+v", record)
	}

	if _, err := st.UpsertProfile(ctx, AgentProfile{
		DID:         "did:key:beta",
		Name:        "Bob",
		Description: "Go backend engineer",
		Skills:      []string{"go", "backend"},
		Tags:        []string{"infra"},
		PeerID:      "12D3KooWBeta",
	}); err != nil {
		t.Fatalf("upsert beta: %v", err)
	}

	resolved, err := st.ResolveANS(ctx, "ALICE")
	if err != nil {
		t.Fatalf("resolve ans: %v", err)
	}
	if resolved.Name != "agent://alice" {
		t.Fatalf("unexpected resolved name: %+v", resolved)
	}

	lookup, err := st.LookupAgents(ctx, []string{"translation", "rust"}, 10)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(lookup) != 1 || lookup[0].DID != "did:key:alpha" {
		t.Fatalf("unexpected lookup results: %+v", lookup)
	}

	discover, err := st.SearchAgents(ctx, "translator", []string{"translation"}, 10)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(discover) != 1 {
		t.Fatalf("unexpected discover length: %+v", discover)
	}
	if discover[0].Name != "agent://alice" {
		t.Fatalf("unexpected discover result: %+v", discover[0])
	}
	if discover[0].Score <= 0 {
		t.Fatalf("expected positive score: %+v", discover[0])
	}
}
