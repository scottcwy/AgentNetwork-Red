package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrConflict            = errors.New("state conflict")
	ErrForbidden           = errors.New("forbidden")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

const DisputeFee int64 = 500

type Task struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	Publisher      string `json:"publisher"`
	Claimant       string `json:"claimant,omitempty"`
	Reward         int64  `json:"reward"`
	Tier           string `json:"tier"`
	Mode           string `json:"mode,omitempty"`
	State          string `json:"state"`
	Result         string `json:"result,omitempty"`
	EscrowAmount   int64  `json:"escrow_amount"`
	DepositAmount  int64  `json:"deposit_amount"`
	Tags           string `json:"tags,omitempty"`
	CreatedAt      string `json:"created_at"`
	ClaimedAt      string `json:"claimed_at,omitempty"`
	SubmittedAt    string `json:"submitted_at,omitempty"`
	SettledAt      string `json:"settled_at,omitempty"`
	StateEnteredAt string `json:"state_entered_at,omitempty"`
}

type BoardStats struct {
	Total   int64            `json:"total"`
	ByState map[string]int64 `json:"by_state"`
}

func FeeForReward(reward int64) int64 {
	return reward * 5 / 100
}

func DepositForReward(reward int64) int64 {
	return reward * 30 / 100
}

func TierForReward(reward int64) string {
	switch {
	case reward >= 5000:
		return "large"
	case reward >= 1500:
		return "medium"
	case reward >= 500:
		return "small"
	default:
		return "micro"
	}
}

func (s *Store) CreateTask(ctx context.Context, task Task) (Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()

	task.CreatedAt = nowRFC3339()
	task.StateEnteredAt = task.CreatedAt
	task.State = "created"
	task.Mode = "simple"
	task.Tier = TierForReward(task.Reward)
	task.EscrowAmount = task.Reward + FeeForReward(task.Reward)
	task.DepositAmount = 0

	if err := applyCreditDeltaTx(ctx, tx, "task:"+task.ID+":create:escrow", task.Publisher, -task.EscrowAmount, "task_create_escrow", task.ID); err != nil {
		return Task{}, err
	}

	if err := insertTaskTx(ctx, tx, task); err != nil {
		return Task{}, err
	}

	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Store) GetTask(ctx context.Context, id string) (Task, error) {
	return getTaskByID(ctx, s.db, id)
}

func (s *Store) ClaimTask(ctx context.Context, id, claimant string) (Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()

	task, err := getTaskByID(ctx, tx, id)
	if err != nil {
		return Task{}, err
	}
	if task.State != "created" {
		return Task{}, ErrConflict
	}
	if task.Publisher == claimant {
		return Task{}, ErrForbidden
	}

	deposit := DepositForReward(task.Reward)
	if err := applyCreditDeltaTx(ctx, tx, "task:"+task.ID+":claim:deposit", claimant, -deposit, "task_claim_deposit", task.ID); err != nil {
		return Task{}, err
	}

	task.Claimant = claimant
	task.DepositAmount = deposit
	task.State = "claimed"
	task.ClaimedAt = nowRFC3339()
	task.StateEnteredAt = task.ClaimedAt

	if err := updateTaskTx(ctx, tx, task); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Store) SubmitTask(ctx context.Context, id, claimant, result string) (Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()

	task, err := getTaskByID(ctx, tx, id)
	if err != nil {
		return Task{}, err
	}
	if task.State != "claimed" {
		return Task{}, ErrConflict
	}
	if task.Claimant != claimant {
		return Task{}, ErrForbidden
	}

	task.State = "submitted"
	task.Result = result
	task.SubmittedAt = nowRFC3339()
	task.StateEnteredAt = task.SubmittedAt

	if err := updateTaskTx(ctx, tx, task); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Store) AcceptTask(ctx context.Context, id, publisher string) (Task, error) {
	return s.settleTask(ctx, id, publisher, "submitted", "accepted")
}

func (s *Store) RejectTask(ctx context.Context, id, publisher string) (Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()

	task, err := getTaskByID(ctx, tx, id)
	if err != nil {
		return Task{}, err
	}
	if task.State != "submitted" {
		return Task{}, ErrConflict
	}
	if task.Publisher != publisher {
		return Task{}, ErrForbidden
	}

	task.State = "rejected"
	task.StateEnteredAt = nowRFC3339()
	if err := updateTaskTx(ctx, tx, task); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Store) DisputeTask(ctx context.Context, id, claimant string) (Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()

	task, err := getTaskByID(ctx, tx, id)
	if err != nil {
		return Task{}, err
	}
	if task.State != "rejected" {
		return Task{}, ErrConflict
	}
	if task.Claimant != claimant {
		return Task{}, ErrForbidden
	}

	if err := applyCreditDeltaTx(ctx, tx, "task:"+task.ID+":dispute:fee", claimant, -DisputeFee, "task_dispute_fee", task.ID); err != nil {
		return Task{}, err
	}

	task.State = "disputed"
	task.StateEnteredAt = nowRFC3339()
	if err := updateTaskTx(ctx, tx, task); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Store) CancelTask(ctx context.Context, id, publisher string) (Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()

	task, err := getTaskByID(ctx, tx, id)
	if err != nil {
		return Task{}, err
	}
	if task.State != "created" {
		return Task{}, ErrConflict
	}
	if task.Publisher != publisher {
		return Task{}, ErrForbidden
	}

	if err := applyCreditDeltaTx(ctx, tx, "task:"+task.ID+":cancel:refund", publisher, task.EscrowAmount, "task_cancel_refund", task.ID); err != nil {
		return Task{}, err
	}

	task.State = "cancelled"
	task.SettledAt = nowRFC3339()
	task.StateEnteredAt = task.SettledAt
	if err := updateTaskTx(ctx, tx, task); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Store) AbandonTask(ctx context.Context, id, claimant string) (Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()

	task, err := getTaskByID(ctx, tx, id)
	if err != nil {
		return Task{}, err
	}
	if task.State != "claimed" {
		return Task{}, ErrConflict
	}
	if task.Claimant != claimant {
		return Task{}, ErrForbidden
	}

	task.State = "created"
	task.Claimant = ""
	task.Result = ""
	task.DepositAmount = 0
	task.ClaimedAt = ""
	task.SubmittedAt = ""
	task.StateEnteredAt = nowRFC3339()
	if err := updateTaskTx(ctx, tx, task); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Store) ArbitrateTask(ctx context.Context, id, publisher, verdict string) (Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()

	task, err := getTaskByID(ctx, tx, id)
	if err != nil {
		return Task{}, err
	}
	if task.State != "disputed" {
		return Task{}, ErrConflict
	}
	if task.Publisher != publisher {
		return Task{}, ErrForbidden
	}

	switch verdict {
	case "favor_claimant":
		task.State = "released"
	case "favor_publisher":
		if err := applyCreditDeltaTx(ctx, tx, "task:"+task.ID+":arbitrate:publisher_refund", publisher, task.Reward, "task_arbitrate_publisher_refund", task.ID); err != nil {
			return Task{}, err
		}
		task.State = "slashed"
	default:
		return Task{}, ErrConflict
	}
	task.SettledAt = nowRFC3339()
	task.StateEnteredAt = task.SettledAt
	if err := updateTaskTx(ctx, tx, task); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Store) UpsertTask(ctx context.Context, task Task) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO tasks(
			id, title, description, publisher, claimant, reward, tier, mode, state, result,
			escrow_amount, deposit_amount, tags, created_at, claimed_at, submitted_at, settled_at, state_entered_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title=excluded.title,
			description=excluded.description,
			publisher=excluded.publisher,
			claimant=excluded.claimant,
			reward=excluded.reward,
			tier=excluded.tier,
			mode=excluded.mode,
			state=excluded.state,
			result=excluded.result,
			escrow_amount=excluded.escrow_amount,
			deposit_amount=excluded.deposit_amount,
			tags=excluded.tags,
			created_at=excluded.created_at,
			claimed_at=excluded.claimed_at,
			submitted_at=excluded.submitted_at,
			settled_at=excluded.settled_at,
			state_entered_at=excluded.state_entered_at`,
		task.ID, task.Title, task.Description, task.Publisher, task.Claimant, task.Reward, task.Tier, task.Mode, task.State, task.Result,
		task.EscrowAmount, task.DepositAmount, task.Tags, task.CreatedAt, task.ClaimedAt, task.SubmittedAt, task.SettledAt, task.StateEnteredAt,
	)
	return err
}

func (s *Store) ListTasks(ctx context.Context, state, query string, limit int) ([]Task, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	args := []any{}
	var where []string
	if state != "" {
		where = append(where, "state = ?")
		args = append(args, state)
	}
	if query != "" {
		where = append(where, "(title LIKE ? OR tags LIKE ?)")
		like := "%" + query + "%"
		args = append(args, like, like)
	}

	stmt := `SELECT id, title, description, publisher, claimant, reward, tier, mode, state, result,
		escrow_amount, deposit_amount, tags, created_at, claimed_at, submitted_at, settled_at, state_entered_at
		FROM tasks`
	if len(where) > 0 {
		stmt += " WHERE " + strings.Join(where, " AND ")
	}
	stmt += " ORDER BY created_at DESC, id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (s *Store) TaskStats(ctx context.Context) (BoardStats, error) {
	stats := BoardStats{ByState: map[string]int64{}}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM tasks`).Scan(&stats.Total); err != nil {
		return BoardStats{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT state, COUNT(1) FROM tasks GROUP BY state`)
	if err != nil {
		return BoardStats{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var state string
		var count int64
		if err := rows.Scan(&state, &count); err != nil {
			return BoardStats{}, err
		}
		stats.ByState[state] = count
	}
	return stats, rows.Err()
}

func (s *Store) ApplyTaskSettlementForSelf(ctx context.Context, task Task, selfDID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	switch task.State {
	case "accepted":
		if task.Claimant == selfDID {
			if err := applyCreditDeltaTx(ctx, tx, "task:"+task.ID+":accepted:claimant", selfDID, task.Reward+task.DepositAmount, "task_accepted_claimant_payout", task.ID); err != nil {
				return err
			}
		}
	case "released":
		if task.Claimant == selfDID {
			if err := applyCreditDeltaTx(ctx, tx, "task:"+task.ID+":released:claimant", selfDID, task.Reward+task.DepositAmount+DisputeFee, "task_released_claimant_payout", task.ID); err != nil {
				return err
			}
		}
	case "slashed":
		if task.Publisher == selfDID {
			if err := applyCreditDeltaTx(ctx, tx, "task:"+task.ID+":slashed:publisher", selfDID, task.Reward, "task_slashed_publisher_refund", task.ID); err != nil {
				return err
			}
		}
	case "cancelled":
		if task.Publisher == selfDID {
			if err := applyCreditDeltaTx(ctx, tx, "task:"+task.ID+":cancel:refund", selfDID, task.EscrowAmount, "task_cancel_refund", task.ID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *Store) settleTask(ctx context.Context, id, publisher, finalState string, wantState string) (Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()

	task, err := getTaskByID(ctx, tx, id)
	if err != nil {
		return Task{}, err
	}
	if task.State != finalState {
		return Task{}, ErrConflict
	}
	if task.Publisher != publisher {
		return Task{}, ErrForbidden
	}

	task.State = wantState
	task.SettledAt = nowRFC3339()
	task.StateEnteredAt = task.SettledAt
	if err := updateTaskTx(ctx, tx, task); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return Task{}, err
	}
	return task, nil
}

type taskScanner interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func getTaskByID(ctx context.Context, q taskScanner, id string) (Task, error) {
	var task Task
	err := q.QueryRowContext(
		ctx,
		`SELECT id, title, description, publisher, claimant, reward, tier, mode, state, result,
			escrow_amount, deposit_amount, tags, created_at, claimed_at, submitted_at, settled_at, state_entered_at
		 FROM tasks WHERE id = ?`,
		id,
	).Scan(
		&task.ID, &task.Title, &task.Description, &task.Publisher, &task.Claimant, &task.Reward, &task.Tier, &task.Mode,
		&task.State, &task.Result, &task.EscrowAmount, &task.DepositAmount, &task.Tags, &task.CreatedAt,
		&task.ClaimedAt, &task.SubmittedAt, &task.SettledAt, &task.StateEnteredAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, err
	}
	return task, nil
}

func insertTaskTx(ctx context.Context, tx *sql.Tx, task Task) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO tasks(
			id, title, description, publisher, claimant, reward, tier, mode, state, result,
			escrow_amount, deposit_amount, tags, created_at, claimed_at, submitted_at, settled_at, state_entered_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Title, task.Description, task.Publisher, task.Claimant, task.Reward, task.Tier, task.Mode, task.State, task.Result,
		task.EscrowAmount, task.DepositAmount, task.Tags, task.CreatedAt, task.ClaimedAt, task.SubmittedAt, task.SettledAt, task.StateEnteredAt,
	)
	return err
}

func updateTaskTx(ctx context.Context, tx *sql.Tx, task Task) error {
	_, err := tx.ExecContext(
		ctx,
		`UPDATE tasks SET
			title = ?, description = ?, publisher = ?, claimant = ?, reward = ?, tier = ?, mode = ?,
			state = ?, result = ?, escrow_amount = ?, deposit_amount = ?, tags = ?, created_at = ?, claimed_at = ?,
			submitted_at = ?, settled_at = ?, state_entered_at = ?
		 WHERE id = ?`,
		task.Title, task.Description, task.Publisher, task.Claimant, task.Reward, task.Tier, task.Mode,
		task.State, task.Result, task.EscrowAmount, task.DepositAmount, task.Tags, task.CreatedAt, task.ClaimedAt,
		task.SubmittedAt, task.SettledAt, task.StateEnteredAt, task.ID,
	)
	return err
}

func scanTasks(rows *sql.Rows) ([]Task, error) {
	var tasks []Task
	for rows.Next() {
		var task Task
		if err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &task.Publisher, &task.Claimant, &task.Reward, &task.Tier, &task.Mode,
			&task.State, &task.Result, &task.EscrowAmount, &task.DepositAmount, &task.Tags, &task.CreatedAt,
			&task.ClaimedAt, &task.SubmittedAt, &task.SettledAt, &task.StateEnteredAt,
		); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func applyCreditDeltaTx(ctx context.Context, tx *sql.Tx, eventID, did string, delta int64, reason, refID string) error {
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM credit_events WHERE id = ?`, eventID).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}

	var balance int64
	err := tx.QueryRowContext(
		ctx,
		`SELECT balance FROM credit_events WHERE peer_did = ? ORDER BY rowid DESC LIMIT 1`,
		did,
	).Scan(&balance)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	newBalance := balance + delta
	if newBalance < 0 {
		return ErrInsufficientBalance
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO credit_events(id, peer_did, delta, balance, reason, ref_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		eventID, did, delta, newBalance, reason, refID, nowRFC3339(),
	)
	return err
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
