package bd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func LoadSubmissions(inboxRoot string) ([]InboxSubmission, error) {
	path := filepath.Join(inboxRoot, SubmissionFileName)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var submissions []InboxSubmission
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var submission InboxSubmission
		if err := json.Unmarshal([]byte(line), &submission); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNumber, err)
		}

		submission.applyDefaults()
		submission.lineNumber = lineNumber
		submissions = append(submissions, submission)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return submissions, nil
}

func SaveSubmissions(inboxRoot string, submissions []InboxSubmission) error {
	if err := EnsureInboxLayout(inboxRoot); err != nil {
		return err
	}
	sortSubmissionsByTime(submissions)

	path := filepath.Join(inboxRoot, SubmissionFileName)
	data := make([]byte, 0, len(submissions)*240)
	for _, submission := range submissions {
		submission.applyDefaults()
		submission.lineNumber = 0
		submission.normalizedProfileURL = ""
		line, err := json.Marshal(submission)
		if err != nil {
			return err
		}
		data = append(data, line...)
		data = append(data, '\n')
	}

	return os.WriteFile(path, data, 0o644)
}

func ValidateSubmission(inboxRoot string, submission InboxSubmission) []string {
	var issues []string
	ref := submission.reference()

	if strings.TrimSpace(submission.SubmissionID) == "" {
		issues = append(issues, fmt.Sprintf("%s: submission_id is required", ref))
	}
	if strings.TrimSpace(submission.ChatID) == "" {
		issues = append(issues, fmt.Sprintf("%s: chat_id is required", ref))
	}
	if strings.TrimSpace(submission.SourceEvent) == "" {
		issues = append(issues, fmt.Sprintf("%s: source_event is required", ref))
	}
	if strings.TrimSpace(submission.XHSProfileURL) == "" {
		issues = append(issues, fmt.Sprintf("%s: xhs_profile_url is required", ref))
	}
	if strings.TrimSpace(submission.CoverPhotoTmp) == "" {
		issues = append(issues, fmt.Sprintf("%s: cover_photo_tmp is required", ref))
	}
	if _, err := time.Parse(time.RFC3339, submission.SubmittedAt); err != nil {
		issues = append(issues, fmt.Sprintf("%s: submitted_at must be RFC3339: %v", ref, err))
	}
	if _, err := NormalizeProfileURL(submission.XHSProfileURL); err != nil {
		issues = append(issues, fmt.Sprintf("%s: invalid xhs_profile_url: %v", ref, err))
	}
	if _, err := resolveInboxFilePath(inboxRoot, submission.CoverPhotoTmp); err != nil {
		issues = append(issues, fmt.Sprintf("%s: %v", ref, err))
	}

	return issues
}

func resolveInboxFilePath(inboxRoot, raw string) (string, error) {
	clean := filepath.ToSlash(strings.TrimSpace(raw))
	if clean == "" {
		return "", fmt.Errorf("cover_photo_tmp is empty")
	}

	var abs string
	if filepath.IsAbs(clean) {
		abs = clean
	} else {
		abs = filepath.Join(filepath.Dir(inboxRoot), filepath.FromSlash(clean))
	}

	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("cover_photo_tmp file does not exist: %s", clean)
	}
	return abs, nil
}

func (s *InboxSubmission) applyDefaults() {
	s.SubmissionID = strings.TrimSpace(s.SubmissionID)
	s.ChatID = strings.TrimSpace(s.ChatID)
	s.Collector = strings.TrimSpace(s.Collector)
	s.SourceEvent = strings.TrimSpace(s.SourceEvent)
	s.XHSProfileURL = strings.TrimSpace(s.XHSProfileURL)
	s.DisplayName = strings.TrimSpace(s.DisplayName)
	s.TeamName = strings.TrimSpace(s.TeamName)
	s.TeamScope = strings.TrimSpace(s.TeamScope)
	s.Note = strings.TrimSpace(s.Note)
	s.CoverPhotoTmp = filepath.ToSlash(strings.TrimSpace(s.CoverPhotoTmp))
	s.SubmittedAt = strings.TrimSpace(s.SubmittedAt)
	s.LastError = strings.TrimSpace(s.LastError)
	s.SyncedAt = strings.TrimSpace(s.SyncedAt)
	if s.Collector == "" {
		s.Collector = DefaultCollector
	}
	if s.State == "" {
		s.State = SubmissionStateReady
	}
}

func (s InboxSubmission) reference() string {
	if s.SubmissionID != "" {
		return s.SubmissionID
	}
	return fmt.Sprintf("line %d", s.lineNumber)
}
