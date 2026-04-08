package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"agentnetwork-red/internal/store"
)

type taskEnvelope struct {
	Action    string     `json:"action"`
	Task      store.Task `json:"task"`
	From      string     `json:"from"`
	Timestamp string     `json:"timestamp"`
}

type taskResponse struct {
	store.Task
	Tip tipInfo `json:"tip,omitempty"`
}

type boardTaskCard struct {
	store.Task
	Tip tipInfo
}

func (d *Daemon) handleTasksCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		d.handleTaskCreate(w, r)
	case http.MethodGet:
		d.handleTaskBoard(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
			Message:    "method not allowed",
			Suggestion: "use GET /api/tasks/board or POST /api/tasks",
		})
	}
}

func (d *Daemon) handleTaskCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Reward      int64  `json:"reward"`
		Tags        string `json:"tags"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Message: err.Error(), Suggestion: "provide title and reward"})
		return
	}
	if strings.TrimSpace(req.Title) == "" || req.Reward <= 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Message: "invalid task payload", Suggestion: "title must be non-empty and reward must be positive"})
		return
	}

	task, err := d.store.CreateTask(r.Context(), store.Task{
		ID:          randomID(),
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Publisher:   d.identity.DID,
		Reward:      req.Reward,
		Tags:        strings.TrimSpace(req.Tags),
	})
	if err != nil {
		d.writeStoreError(w, err, "create task")
		return
	}
	d.afterTaskMutation(r.Context(), "created", task)
	writeJSON(w, http.StatusOK, task)
}

func (d *Daemon) handleTaskItem(w http.ResponseWriter, r *http.Request) {
	id, action, err := parseTaskPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Message: err.Error(), Suggestion: "use /api/tasks/{id} or /api/tasks/{id}/{action}"})
		return
	}

	if action == "" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Message: "method not allowed", Suggestion: "use GET /api/tasks/{id}"})
			return
		}
		task, err := d.store.GetTask(r.Context(), id)
		if err != nil {
			d.writeStoreError(w, err, "fetch task")
			return
		}
		writeJSON(w, http.StatusOK, task)
		return
	}

	if action == "bundle" {
		switch r.Method {
		case http.MethodGet:
			d.handleTaskBundleGet(w, r, id)
		case http.MethodPost:
			d.handleTaskBundlePost(w, r, id)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Message: "method not allowed", Suggestion: "use GET or POST /api/tasks/{id}/bundle"})
		}
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Message: "method not allowed", Suggestion: "use POST for task actions"})
		return
	}

	switch action {
	case "claim":
		task, err := d.store.ClaimTask(r.Context(), id, d.identity.DID)
		if err != nil {
			d.writeStoreError(w, err, "claim task")
			return
		}
		d.afterTaskMutation(r.Context(), "claimed", task)
		writeJSON(w, http.StatusOK, task)
	case "submit":
		var req struct {
			Result string `json:"result"`
		}
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Message: err.Error(), Suggestion: "provide result text"})
			return
		}
		task, err := d.store.SubmitTask(r.Context(), id, d.identity.DID, req.Result)
		if err != nil {
			d.writeStoreError(w, err, "submit task")
			return
		}
		d.afterTaskMutation(r.Context(), "submitted", task)
		writeJSON(w, http.StatusOK, task)
	case "accept":
		task, err := d.store.AcceptTask(r.Context(), id, d.identity.DID)
		if err != nil {
			d.writeStoreError(w, err, "accept task")
			return
		}
		d.afterTaskMutation(r.Context(), "accepted", task)
		writeJSON(w, http.StatusOK, taskResponse{
			Task: task,
			Tip:  d.resolveTipInfo(r.Context(), task.Claimant),
		})
	case "reject":
		task, err := d.store.RejectTask(r.Context(), id, d.identity.DID)
		if err != nil {
			d.writeStoreError(w, err, "reject task")
			return
		}
		d.afterTaskMutation(r.Context(), "rejected", task)
		writeJSON(w, http.StatusOK, task)
	case "dispute":
		task, err := d.store.DisputeTask(r.Context(), id, d.identity.DID)
		if err != nil {
			d.writeStoreError(w, err, "dispute task")
			return
		}
		d.afterTaskMutation(r.Context(), "disputed", task)
		writeJSON(w, http.StatusOK, task)
	case "cancel":
		task, err := d.store.CancelTask(r.Context(), id, d.identity.DID)
		if err != nil {
			d.writeStoreError(w, err, "cancel task")
			return
		}
		d.afterTaskMutation(r.Context(), "cancelled", task)
		writeJSON(w, http.StatusOK, task)
	case "abandon":
		task, err := d.store.AbandonTask(r.Context(), id, d.identity.DID)
		if err != nil {
			d.writeStoreError(w, err, "abandon task")
			return
		}
		d.afterTaskMutation(r.Context(), "abandoned", task)
		writeJSON(w, http.StatusOK, task)
	case "arbitrate":
		var req struct {
			Verdict string `json:"verdict"`
		}
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Message: err.Error(), Suggestion: "provide verdict"})
			return
		}
		task, err := d.store.ArbitrateTask(r.Context(), id, d.identity.DID, req.Verdict)
		if err != nil {
			d.writeStoreError(w, err, "arbitrate task")
			return
		}
		d.afterTaskMutation(r.Context(), task.State, task)
		writeJSON(w, http.StatusOK, task)
	default:
		writeJSON(w, http.StatusNotFound, errorResponse{Message: "unknown task action", Suggestion: "use claim, submit, accept, reject, dispute, cancel, abandon, or arbitrate"})
	}
}

func (d *Daemon) handleTaskBundleGet(w http.ResponseWriter, r *http.Request, id string) {
	bundle, err := d.store.GetTaskBundle(r.Context(), id)
	if err != nil {
		d.writeStoreError(w, err, "fetch task bundle")
		return
	}

	filename := bundle.Filename
	if filename == "" {
		filename = id + ".nut"
	}
	w.Header().Set("Content-Type", bundle.MimeType)
	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bundle.Data)
}

func (d *Daemon) handleTaskBundlePost(w http.ResponseWriter, r *http.Request, id string) {
	task, err := d.store.GetTask(r.Context(), id)
	if err != nil {
		d.writeStoreError(w, err, "fetch task")
		return
	}
	if task.Publisher != d.identity.DID && task.Claimant != d.identity.DID {
		writeJSON(w, http.StatusForbidden, errorResponse{
			Message:    "forbidden",
			Suggestion: "only the publisher or claimant can attach a bundle",
		})
		return
	}

	data, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Message: err.Error(), Suggestion: "send the bundle as a raw request body"})
		return
	}
	if len(data) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Message: "empty bundle payload", Suggestion: "send a non-empty .nut payload"})
		return
	}

	filename := strings.TrimSpace(r.Header.Get("X-Filename"))
	if filename == "" {
		if raw := r.URL.Query().Get("filename"); raw != "" {
			if decoded, decodeErr := url.QueryUnescape(raw); decodeErr == nil {
				filename = strings.TrimSpace(decoded)
			}
		}
	}
	if filename == "" {
		filename = id + ".nut"
	}

	mimeType := strings.TrimSpace(r.Header.Get("Content-Type"))
	if mimeType == "" || mimeType == "application/x-www-form-urlencoded" {
		mimeType = mime.TypeByExtension(filepath.Ext(filename))
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	bundle := store.TaskBundle{
		TaskID:     id,
		Filename:   filename,
		MimeType:   mimeType,
		Data:       data,
		Uploader:   d.identity.DID,
		UploadedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := d.store.UpsertTaskBundle(r.Context(), bundle); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:    err.Error(),
			Suggestion: "check the local bundle store health",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"task_id":     bundle.TaskID,
		"filename":    bundle.Filename,
		"mime_type":   bundle.MimeType,
		"uploader":    bundle.Uploader,
		"uploaded_at": bundle.UploadedAt,
		"size_bytes":  len(bundle.Data),
	})
}

func (d *Daemon) handleTaskBoard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Message: "method not allowed", Suggestion: "use GET /api/tasks/board"})
		return
	}
	limit := queryInt(r, "limit", 50)
	tasks, err := d.store.ListTasks(r.Context(), r.URL.Query().Get("state"), r.URL.Query().Get("q"), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Message: err.Error(), Suggestion: "check the local task store"})
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (d *Daemon) handleTaskBoardStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Message: "method not allowed", Suggestion: "use GET /api/tasks/board/stats"})
		return
	}
	stats, err := d.store.TaskStats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Message: err.Error(), Suggestion: "check the local task store"})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (d *Daemon) handleBoardUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tasks, err := d.store.ListTasks(r.Context(), "", "", 200)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	columns := []string{"created", "claimed", "submitted", "accepted", "disputed"}
	grouped := map[string][]boardTaskCard{}
	for _, task := range tasks {
		card := boardTaskCard{Task: task}
		if task.Claimant != "" && (task.State == "accepted" || task.State == "released") {
			card.Tip = d.resolveTipInfo(r.Context(), task.Claimant)
		}
		grouped[task.State] = append(grouped[task.State], card)
	}
	tmpl := template.Must(template.New("board").Parse(boardHTML))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, map[string]any{
		"Columns": columns,
		"Tasks":   grouped,
	})
}

func (d *Daemon) receiveTaskPayload(ctx context.Context, payload []byte) error {
	var envelope taskEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return err
	}
	if envelope.Task.ID == "" {
		return errors.New("invalid task payload")
	}
	if err := d.store.UpsertTask(ctx, envelope.Task); err != nil {
		return err
	}
	return d.store.ApplyTaskSettlementForSelf(ctx, envelope.Task, d.identity.DID)
}

func (d *Daemon) afterTaskMutation(ctx context.Context, action string, task store.Task) {
	if err := d.store.ApplyTaskSettlementForSelf(ctx, task, d.identity.DID); err != nil {
		// Keep the mutation successful even if the local derived credit sync fails.
		// This path is idempotent and can be retried from the broadcast event.
		log.Printf("task settlement side effect error task=%s err=%v", task.ID, err)
	}
	if d.p2p == nil {
		return
	}
	payload, err := json.Marshal(taskEnvelope{
		Action:    action,
		Task:      task,
		From:      d.identity.DID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return
	}

	publish := func(delay time.Duration) {
		go func() {
			if delay > 0 {
				time.Sleep(delay)
			}
			publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := d.p2p.PublishTask(publishCtx, payload); err != nil {
				log.Printf("task publish error task=%s err=%v", task.ID, err)
			}
		}()
	}
	publish(0)
	publish(500 * time.Millisecond)
	publish(1500 * time.Millisecond)
}

func (d *Daemon) writeStoreError(w http.ResponseWriter, err error, action string) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Message: err.Error(), Suggestion: action + ": task not found"})
	case errors.Is(err, store.ErrConflict):
		writeJSON(w, http.StatusConflict, errorResponse{Message: err.Error(), Suggestion: action + ": invalid state transition"})
	case errors.Is(err, store.ErrForbidden):
		writeJSON(w, http.StatusForbidden, errorResponse{Message: err.Error(), Suggestion: action + ": caller is not allowed to do that"})
	case errors.Is(err, store.ErrInsufficientBalance):
		writeJSON(w, http.StatusConflict, errorResponse{Message: err.Error(), Suggestion: action + ": top up local credits first"})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Message: err.Error(), Suggestion: "retry after checking the local store"})
	}
}

func parseTaskPath(path string) (id string, action string, err error) {
	const prefix = "/api/tasks/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", errors.New("invalid task path")
	}
	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", "", errors.New("missing task id")
	}
	id = parts[0]
	if len(parts) > 1 {
		action = parts[1]
	}
	return id, action, nil
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

const boardHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>RedAnet Board</title>
<style>
body{font-family:ui-sans-serif,system-ui,sans-serif;background:#f5f3ee;color:#16130f;margin:0;padding:24px}
.board{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:16px}
.col{background:#fff9ef;border:1px solid #e4d8c7;border-radius:16px;padding:16px;box-shadow:0 10px 30px rgba(0,0,0,.04)}
.card{background:white;border:1px solid #eadfce;border-radius:12px;padding:12px;margin:12px 0}
.state{font-size:12px;color:#8a7356;text-transform:uppercase;letter-spacing:.08em}
h1{margin:0 0 16px}
h2{margin:0 0 8px;font-size:18px}
p{margin:6px 0}
button{border:0;border-radius:999px;background:#d94f1d;color:#fff;padding:8px 12px;cursor:pointer}
dialog{border:0;border-radius:16px;padding:0;max-width:min(92vw,420px)}
dialog::backdrop{background:rgba(20,16,12,.45)}
.tip-panel{padding:18px;background:#fff8ef}
.tip-panel img{display:block;max-width:100%;border-radius:12px;border:1px solid #eadfce}
.tip-panel form{margin-top:12px;text-align:right}
</style>
</head>
<body>
<h1>RedAnet Board</h1>
<div class="board">
{{range .Columns}}
<section class="col">
<h2>{{.}}</h2>
{{range index $.Tasks .}}
<article class="card">
<div class="state">{{.State}}</div>
<p><strong>{{.Title}}</strong></p>
<p>reward: {{.Reward}}</p>
<p>{{.Description}}</p>
{{if .Tip.Available}}
<button type="button" onclick="openTip('{{.ID}}','{{.Tip.QRCodeURL}}','{{.Tip.ClaimantDID}}')">打赏</button>
{{end}}
</article>
{{else}}
<p>empty</p>
{{end}}
</section>
{{end}}
</div>
<dialog id="tip-dialog">
  <div class="tip-panel">
    <p id="tip-title"></p>
    <img id="tip-image" alt="tip qrcode">
    <form method="dialog">
      <button type="submit">关闭</button>
    </form>
  </div>
</dialog>
<script>
const tipDialog = document.getElementById('tip-dialog');
const tipTitle = document.getElementById('tip-title');
const tipImage = document.getElementById('tip-image');
function openTip(taskID, url, claimantDID) {
  tipTitle.textContent = '任务 ' + taskID + ' · ' + claimantDID;
  tipImage.src = url;
  tipDialog.showModal();
}
</script>
</body>
</html>`
