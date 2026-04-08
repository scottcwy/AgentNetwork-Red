package bd

const (
	DefaultCollector = "Zeena"
	SeedFileName     = "xhs_seeds.jsonl"
	InboxDirName     = ".bd-inbox"

	DraftsDirName      = "drafts"
	SubmissionFileName = "submissions.jsonl"
	InboxFilesDirName  = "files"
)

type SeedRecord struct {
	SeedID        string `json:"seed_id"`
	XHSProfileURL string `json:"xhs_profile_url"`
	CoverPhoto    string `json:"cover_photo"`
	SourceEvent   string `json:"source_event"`
	CapturedAt    string `json:"captured_at"`
	Collector     string `json:"collector,omitempty"`
	DisplayName   string `json:"display_name,omitempty"`
	TeamName      string `json:"team_name,omitempty"`
	TeamScope     string `json:"team_scope,omitempty"`
	Note          string `json:"note,omitempty"`

	lineNumber           int
	normalizedProfileURL string
}

type RepositorySummary struct {
	SeedCount  int
	PhotoCount int
}

type DisplaySummary struct {
	GeneratedAt string
	SeedCount   int
	EventCount  int
}

type DraftRecord struct {
	ChatID        string `json:"chat_id"`
	Collector     string `json:"collector,omitempty"`
	SourceEvent   string `json:"source_event,omitempty"`
	XHSProfileURL string `json:"xhs_profile_url,omitempty"`
	DisplayName   string `json:"display_name,omitempty"`
	TeamName      string `json:"team_name,omitempty"`
	TeamScope     string `json:"team_scope,omitempty"`
	Note          string `json:"note,omitempty"`
	PendingPhoto  string `json:"pending_photo,omitempty"`
	UpdatedAt     string `json:"updated_at"`
}

type SubmissionState string

const (
	SubmissionStateReady       SubmissionState = "ready"
	SubmissionStateSynced      SubmissionState = "synced"
	SubmissionStateNeedsReview SubmissionState = "needs_review"
	SubmissionStateError       SubmissionState = "error"
)

type InboxSubmission struct {
	SubmissionID  string          `json:"submission_id"`
	ChatID        string          `json:"chat_id"`
	Collector     string          `json:"collector,omitempty"`
	SourceEvent   string          `json:"source_event"`
	XHSProfileURL string          `json:"xhs_profile_url"`
	DisplayName   string          `json:"display_name,omitempty"`
	TeamName      string          `json:"team_name,omitempty"`
	TeamScope     string          `json:"team_scope,omitempty"`
	Note          string          `json:"note,omitempty"`
	CoverPhotoTmp string          `json:"cover_photo_tmp"`
	SubmittedAt   string          `json:"submitted_at"`
	State         SubmissionState `json:"state,omitempty"`
	LastError     string          `json:"last_error,omitempty"`
	SyncedAt      string          `json:"synced_at,omitempty"`

	lineNumber           int
	normalizedProfileURL string
}

type SyncSummary struct {
	Processed     int
	Created       int
	Updated       int
	NeedsReview   int
	Errors        int
	Skipped       int
	GeneratedWrap bool
}
