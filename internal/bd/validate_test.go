package bd

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestValidateRepositorySuccess(t *testing.T) {
	root := writeSeedRepository(t, []string{
		`{"seed_id":"seed-20260408-001","xhs_profile_url":"https://www.xiaohongshu.com/user/profile/sample_linwei","cover_photo":"bd/photos/seed-20260408-001-cover.svg","source_event":"shanghai-demo-day","captured_at":"2026-04-08T12:05:00Z"}`,
	})

	summary, issues := ValidateRepository(root)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
	if summary.SeedCount != 1 {
		t.Fatalf("expected one seed, got %d", summary.SeedCount)
	}
	if summary.PhotoCount != 1 {
		t.Fatalf("expected one photo, got %d", summary.PhotoCount)
	}
}

func TestValidateRepositoryRejectsDuplicateNormalizedURL(t *testing.T) {
	root := writeSeedRepository(t, []string{
		`{"seed_id":"seed-20260408-001","xhs_profile_url":"https://www.xiaohongshu.com/user/profile/sample_linwei","cover_photo":"bd/photos/seed-20260408-001-cover.svg","source_event":"shanghai-demo-day","captured_at":"2026-04-08T12:05:00Z"}`,
		`{"seed_id":"seed-20260408-002","xhs_profile_url":"https://www.xiaohongshu.com/user/profile/sample_linwei/?foo=bar","cover_photo":"bd/photos/seed-20260408-002-cover.svg","source_event":"shanghai-demo-day","captured_at":"2026-04-08T12:15:00Z"}`,
	})

	_, issues := ValidateRepository(root)
	assertIssueContains(t, issues, "normalized profile URL duplicates")
}

func TestLoadSeedsDefaultsCollector(t *testing.T) {
	root := writeSeedRepository(t, []string{
		`{"seed_id":"seed-20260408-001","xhs_profile_url":"https://www.xiaohongshu.com/user/profile/sample_linwei","cover_photo":"bd/photos/seed-20260408-001-cover.svg","source_event":"shanghai-demo-day","captured_at":"2026-04-08T12:05:00Z"}`,
	})

	seeds, err := LoadSeeds(root)
	if err != nil {
		t.Fatalf("load seeds: %v", err)
	}
	if seeds[0].Collector != DefaultCollector {
		t.Fatalf("expected default collector %q, got %q", DefaultCollector, seeds[0].Collector)
	}
}

func TestGenerateWrapUpWritesHTML(t *testing.T) {
	root := writeSeedRepository(t, []string{
		`{"seed_id":"seed-20260408-001","xhs_profile_url":"https://www.xiaohongshu.com/user/profile/sample_linwei","cover_photo":"bd/photos/seed-20260408-001-cover.svg","source_event":"shanghai-demo-day","captured_at":"2026-04-08T12:05:00Z","display_name":"示例-林未","team_name":"虾网探针组","team_scope":"做线下种子沉淀。","note":"现场补充了团队方向。"}`,
	})

	output := filepath.Join(root, "display", "index.html")
	if err := GenerateWrapUp(root, output); err != nil {
		t.Fatalf("generate wrap-up: %v", err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read wrap output: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "示例-林未") {
		t.Fatalf("expected display name in output")
	}
	if !strings.Contains(content, DefaultCollector) {
		t.Fatalf("expected default collector in output")
	}
	if !strings.Contains(content, "../photos/seed-20260408-001-cover.svg") {
		t.Fatalf("expected relative photo path in output")
	}
}

func TestRawFirstSchemaSQLApplies(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime caller unavailable")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	schemaPath := filepath.Join(repoRoot, "bd", "db-schema.sql")
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema sql: %v", err)
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(string(schemaSQL)); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
}

func TestSyncInboxCreatesSeedAndWrapUp(t *testing.T) {
	repoRoot := t.TempDir()
	bdRoot := filepath.Join(repoRoot, "bd")
	inboxRoot := filepath.Join(repoRoot, InboxDirName)

	mkdirAll(t, filepath.Join(bdRoot, "photos"))
	mustWriteFile(t, filepath.Join(bdRoot, SeedFileName), "")
	mkdirAll(t, filepath.Join(inboxRoot, InboxFilesDirName))

	photoPath := filepath.Join(inboxRoot, InboxFilesDirName, "sub-20260408-001.jpg")
	mustWriteFile(t, photoPath, "seed-photo")

	submissions := []InboxSubmission{
		{
			SubmissionID:  "sub-20260408-001",
			ChatID:        "oc_001",
			SourceEvent:   "shanghai-demo-day",
			XHSProfileURL: "https://www.xiaohongshu.com/user/profile/sample_linwei",
			DisplayName:   "示例-林未",
			TeamName:      "虾网探针组",
			TeamScope:     "做线下种子沉淀。",
			CoverPhotoTmp: filepath.ToSlash(photoPath),
			SubmittedAt:   "2026-04-08T12:05:00Z",
		},
	}
	if err := SaveSubmissions(inboxRoot, submissions); err != nil {
		t.Fatalf("save submissions: %v", err)
	}

	summary, err := SyncInbox(bdRoot, inboxRoot, filepath.Join(bdRoot, "display", "index.html"))
	if err != nil {
		t.Fatalf("sync inbox: %v", err)
	}
	if summary.Created != 1 {
		t.Fatalf("expected one created seed, got %+v", summary)
	}
	if !summary.GeneratedWrap {
		t.Fatalf("expected wrap generation")
	}

	seeds, err := LoadSeeds(bdRoot)
	if err != nil {
		t.Fatalf("load seeds: %v", err)
	}
	if len(seeds) != 1 {
		t.Fatalf("expected one seed, got %d", len(seeds))
	}
	if seeds[0].Collector != DefaultCollector {
		t.Fatalf("expected default collector, got %q", seeds[0].Collector)
	}
	if seeds[0].DisplayName != "示例-林未" {
		t.Fatalf("expected display name to be synced")
	}
	if seeds[0].CoverPhoto != "bd/photos/seed-20260408-001-cover.jpg" {
		t.Fatalf("unexpected cover photo path %q", seeds[0].CoverPhoto)
	}

	synced, err := LoadSubmissions(inboxRoot)
	if err != nil {
		t.Fatalf("load synced submissions: %v", err)
	}
	if synced[0].State != SubmissionStateSynced {
		t.Fatalf("expected submission to be synced, got %q", synced[0].State)
	}
	if _, err := os.Stat(filepath.Join(bdRoot, "display", "index.html")); err != nil {
		t.Fatalf("wrap output missing: %v", err)
	}
}

func TestSyncInboxMarksConflictingFieldsForReview(t *testing.T) {
	repoRoot := t.TempDir()
	bdRoot := filepath.Join(repoRoot, "bd")
	inboxRoot := filepath.Join(repoRoot, InboxDirName)

	mkdirAll(t, filepath.Join(bdRoot, "photos"))
	mustWriteFile(t, filepath.Join(bdRoot, SeedFileName), strings.Join([]string{
		`{"seed_id":"seed-20260408-001","xhs_profile_url":"https://www.xiaohongshu.com/user/profile/sample_linwei","cover_photo":"bd/photos/seed-20260408-001-cover.jpg","source_event":"shanghai-demo-day","captured_at":"2026-04-08T12:05:00Z","display_name":"旧名字","team_name":"虾网探针组","team_scope":"旧 scope","note":"旧 note"}`,
		"",
	}, "\n"))
	mustWriteFile(t, filepath.Join(bdRoot, "photos", "seed-20260408-001-cover.jpg"), "existing-photo")
	mkdirAll(t, filepath.Join(inboxRoot, InboxFilesDirName))

	photoPath := filepath.Join(inboxRoot, InboxFilesDirName, "sub-20260408-002.jpg")
	mustWriteFile(t, photoPath, "new-photo")

	submissions := []InboxSubmission{
		{
			SubmissionID:  "sub-20260408-002",
			ChatID:        "oc_001",
			SourceEvent:   "shanghai-demo-day",
			XHSProfileURL: "https://www.xiaohongshu.com/user/profile/sample_linwei?foo=bar",
			DisplayName:   "新名字",
			TeamName:      "虾网探针组",
			TeamScope:     "旧 scope",
			Note:          "旧 note",
			CoverPhotoTmp: filepath.ToSlash(photoPath),
			SubmittedAt:   "2026-04-08T12:15:00Z",
		},
	}
	if err := SaveSubmissions(inboxRoot, submissions); err != nil {
		t.Fatalf("save submissions: %v", err)
	}

	summary, err := SyncInbox(bdRoot, inboxRoot, filepath.Join(bdRoot, "display", "index.html"))
	if err != nil {
		t.Fatalf("sync inbox: %v", err)
	}
	if summary.NeedsReview != 1 {
		t.Fatalf("expected one needs_review submission, got %+v", summary)
	}

	synced, err := LoadSubmissions(inboxRoot)
	if err != nil {
		t.Fatalf("load synced submissions: %v", err)
	}
	if synced[0].State != SubmissionStateNeedsReview {
		t.Fatalf("expected needs_review, got %q", synced[0].State)
	}

	seeds, err := LoadSeeds(bdRoot)
	if err != nil {
		t.Fatalf("load seeds: %v", err)
	}
	if len(seeds) != 1 {
		t.Fatalf("expected one seed, got %d", len(seeds))
	}
	if seeds[0].DisplayName != "旧名字" {
		t.Fatalf("expected existing seed to remain unchanged")
	}
}

func writeSeedRepository(t *testing.T, lines []string) string {
	t.Helper()

	repoRoot := t.TempDir()
	root := filepath.Join(repoRoot, "bd")
	mkdirAll(t, filepath.Join(root, "photos"))

	var builder strings.Builder
	for _, line := range lines {
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	mustWriteFile(t, filepath.Join(root, SeedFileName), builder.String())

	for index := range lines {
		fileName := filepath.Join(root, "photos", seedPhotoName(index+1))
		mustWriteFile(t, fileName, "<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>")
	}

	return root
}

func seedPhotoName(n int) string {
	return fmt.Sprintf("seed-20260408-%03d-cover.svg", n)
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func assertIssueContains(t *testing.T, issues []string, needle string) {
	t.Helper()
	for _, issue := range issues {
		if strings.Contains(issue, needle) {
			return
		}
	}
	t.Fatalf("expected issue containing %q, got %v", needle, issues)
}
