package store

import (
	"context"
	"testing"
)

func TestTaskLifecycleAndCredits(t *testing.T) {
	t.Parallel()

	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	publisher := "did:key:publisher"
	claimant := "did:key:claimant"

	if err := st.EnsureInitialBalance(ctx, publisher, InitialBalance); err != nil {
		t.Fatal(err)
	}
	if err := st.EnsureInitialBalance(ctx, claimant, InitialBalance); err != nil {
		t.Fatal(err)
	}

	task, err := st.CreateTask(ctx, Task{
		ID:          "task-1",
		Title:       "Write tests",
		Description: "Add regression coverage",
		Publisher:   publisher,
		Reward:      500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.State != "created" || task.EscrowAmount != 525 {
		t.Fatalf("unexpected created task: %+v", task)
	}

	publisherBalance, err := st.Balance(ctx, publisher)
	if err != nil {
		t.Fatal(err)
	}
	if publisherBalance != 9475 {
		t.Fatalf("unexpected publisher balance after create: %d", publisherBalance)
	}

	task, err = st.ClaimTask(ctx, task.ID, claimant)
	if err != nil {
		t.Fatal(err)
	}
	if task.State != "claimed" || task.DepositAmount != 150 {
		t.Fatalf("unexpected claimed task: %+v", task)
	}

	claimantBalance, err := st.Balance(ctx, claimant)
	if err != nil {
		t.Fatal(err)
	}
	if claimantBalance != 9850 {
		t.Fatalf("unexpected claimant balance after claim: %d", claimantBalance)
	}

	task, err = st.SubmitTask(ctx, task.ID, claimant, "done")
	if err != nil {
		t.Fatal(err)
	}
	if task.State != "submitted" {
		t.Fatalf("unexpected submit state: %s", task.State)
	}

	task, err = st.AcceptTask(ctx, task.ID, publisher)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ApplyTaskSettlementForSelf(ctx, task, claimant); err != nil {
		t.Fatal(err)
	}

	claimantBalance, err = st.Balance(ctx, claimant)
	if err != nil {
		t.Fatal(err)
	}
	if claimantBalance != 10500 {
		t.Fatalf("unexpected claimant balance after accept: %d", claimantBalance)
	}

	stats, err := st.TaskStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 1 || stats.ByState["accepted"] != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
