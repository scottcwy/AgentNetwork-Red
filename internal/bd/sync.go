package bd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func SyncInbox(bdRoot, inboxRoot, wrapOutputPath string) (SyncSummary, error) {
	var summary SyncSummary

	if err := EnsureInboxLayout(inboxRoot); err != nil {
		return summary, err
	}

	seeds, err := LoadSeeds(bdRoot)
	if err != nil {
		return summary, err
	}
	submissions, err := LoadSubmissions(inboxRoot)
	if err != nil {
		return summary, err
	}

	seedIndex, err := buildSeedIndex(seeds)
	if err != nil {
		return summary, err
	}

	for i := range submissions {
		submission := &submissions[i]
		submission.applyDefaults()

		switch submission.State {
		case SubmissionStateSynced, SubmissionStateNeedsReview:
			summary.Skipped++
			continue
		case SubmissionStateError, SubmissionStateReady:
		default:
			submission.State = SubmissionStateReady
		}

		summary.Processed++
		if issues := ValidateSubmission(inboxRoot, *submission); len(issues) > 0 {
			submission.State = SubmissionStateError
			submission.LastError = strings.Join(issues, "; ")
			submission.SyncedAt = ""
			summary.Errors++
			continue
		}

		normalizedURL, _ := NormalizeProfileURL(submission.XHSProfileURL)
		submission.normalizedProfileURL = normalizedURL

		seedIdx, exists := seedIndex[normalizedURL]
		if exists {
			seed := &seeds[seedIdx]
			conflicts := mergeOptionalFields(seed, *submission)
			if len(conflicts) > 0 {
				submission.State = SubmissionStateNeedsReview
				submission.LastError = "conflicting fields: " + strings.Join(conflicts, ", ")
				submission.SyncedAt = ""
				summary.NeedsReview++
				continue
			}

			relativePhoto, err := copySubmissionPhoto(filepath.Dir(bdRoot), inboxRoot, submission, seed.SeedID)
			if err != nil {
				submission.State = SubmissionStateError
				submission.LastError = err.Error()
				submission.SyncedAt = ""
				summary.Errors++
				continue
			}
			seed.CoverPhoto = relativePhoto

			submission.State = SubmissionStateSynced
			submission.LastError = ""
			submission.SyncedAt = NowRFC3339()
			summary.Updated++
			continue
		}

		seedID, err := nextSeedID(seeds, submission.SubmittedAt)
		if err != nil {
			submission.State = SubmissionStateError
			submission.LastError = err.Error()
			submission.SyncedAt = ""
			summary.Errors++
			continue
		}

		relativePhoto, err := copySubmissionPhoto(filepath.Dir(bdRoot), inboxRoot, submission, seedID)
		if err != nil {
			submission.State = SubmissionStateError
			submission.LastError = err.Error()
			submission.SyncedAt = ""
			summary.Errors++
			continue
		}

		newSeed := SeedRecord{
			SeedID:        seedID,
			XHSProfileURL: normalizedURL,
			CoverPhoto:    relativePhoto,
			SourceEvent:   submission.SourceEvent,
			CapturedAt:    submission.SubmittedAt,
			Collector:     submission.Collector,
			DisplayName:   submission.DisplayName,
			TeamName:      submission.TeamName,
			TeamScope:     submission.TeamScope,
			Note:          submission.Note,
		}
		newSeed.applyDefaults()
		seeds = append(seeds, newSeed)
		seedIndex[normalizedURL] = len(seeds) - 1

		submission.State = SubmissionStateSynced
		submission.LastError = ""
		submission.SyncedAt = NowRFC3339()
		summary.Created++
	}

	if issues := ValidateSeeds(bdRoot, seeds); len(issues) > 0 {
		return summary, fmt.Errorf("seed validation failed after sync:\n- %s", joinIssues(issues))
	}
	if err := SaveSeeds(bdRoot, seeds); err != nil {
		return summary, err
	}
	if err := SaveSubmissions(inboxRoot, submissions); err != nil {
		return summary, err
	}
	if err := GenerateWrapUp(bdRoot, wrapOutputPath); err != nil {
		return summary, err
	}
	summary.GeneratedWrap = true

	return summary, nil
}

func buildSeedIndex(seeds []SeedRecord) (map[string]int, error) {
	index := make(map[string]int, len(seeds))
	for i := range seeds {
		normalized, err := NormalizeProfileURL(seeds[i].XHSProfileURL)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", seeds[i].reference(), err)
		}
		seeds[i].normalizedProfileURL = normalized
		index[normalized] = i
	}
	return index, nil
}

func mergeOptionalFields(seed *SeedRecord, submission InboxSubmission) []string {
	var conflicts []string

	if seed.DisplayName == "" {
		seed.DisplayName = submission.DisplayName
	} else if submission.DisplayName != "" && submission.DisplayName != seed.DisplayName {
		conflicts = append(conflicts, "display_name")
	}

	if seed.TeamName == "" {
		seed.TeamName = submission.TeamName
	} else if submission.TeamName != "" && submission.TeamName != seed.TeamName {
		conflicts = append(conflicts, "team_name")
	}

	if seed.TeamScope == "" {
		seed.TeamScope = submission.TeamScope
	} else if submission.TeamScope != "" && submission.TeamScope != seed.TeamScope {
		conflicts = append(conflicts, "team_scope")
	}

	if seed.Note == "" {
		seed.Note = submission.Note
	} else if submission.Note != "" && submission.Note != seed.Note {
		conflicts = append(conflicts, "note")
	}

	if seed.Collector == "" {
		seed.Collector = submission.Collector
	}

	return conflicts
}

func copySubmissionPhoto(repoRoot, inboxRoot string, submission *InboxSubmission, seedID string) (string, error) {
	sourcePath, err := resolveInboxFilePath(inboxRoot, submission.CoverPhotoTmp)
	if err != nil {
		return "", err
	}

	ext := strings.ToLower(filepath.Ext(sourcePath))
	if ext == "" {
		ext = ".jpg"
	}
	relativeTarget := filepath.ToSlash(filepath.Join("bd", "photos", seedID+"-cover"+ext))
	targetPath := filepath.Join(repoRoot, filepath.FromSlash(relativeTarget))

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", err
	}
	if err := copyFile(sourcePath, targetPath); err != nil {
		return "", err
	}

	submission.CoverPhotoTmp = filepath.ToSlash(submission.CoverPhotoTmp)
	return relativeTarget, nil
}

func copyFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Close()
}

func nextSeedID(seeds []SeedRecord, submittedAt string) (string, error) {
	ts, err := time.Parse(time.RFC3339, submittedAt)
	if err != nil {
		return "", fmt.Errorf("invalid submitted_at for seed id generation: %w", err)
	}

	prefix := "seed-" + ts.UTC().Format("20060102") + "-"
	maxSeq := 0
	for _, seed := range seeds {
		if !strings.HasPrefix(seed.SeedID, prefix) {
			continue
		}
		var seq int
		if _, err := fmt.Sscanf(seed.SeedID, prefix+"%03d", &seq); err == nil && seq > maxSeq {
			maxSeq = seq
		}
	}

	return fmt.Sprintf("%s%03d", prefix, maxSeq+1), nil
}

func sortSubmissionsByTime(submissions []InboxSubmission) {
	sort.SliceStable(submissions, func(i, j int) bool {
		left, errLeft := time.Parse(time.RFC3339, submissions[i].SubmittedAt)
		right, errRight := time.Parse(time.RFC3339, submissions[j].SubmittedAt)
		if errLeft != nil || errRight != nil {
			return submissions[i].SubmittedAt < submissions[j].SubmittedAt
		}
		return left.Before(right)
	})
}
