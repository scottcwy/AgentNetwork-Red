package bd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var seedIDPattern = regexp.MustCompile(`^seed-\d{8}-\d{3}$`)

func LoadSeeds(root string) ([]SeedRecord, error) {
	file, err := os.Open(filepath.Join(root, SeedFileName))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var seeds []SeedRecord
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var seed SeedRecord
		if err := json.Unmarshal([]byte(line), &seed); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNumber, err)
		}

		seed.applyDefaults()
		seed.lineNumber = lineNumber
		seeds = append(seeds, seed)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return seeds, nil
}

func ValidateRepository(root string) (RepositorySummary, []string) {
	var summary RepositorySummary

	seeds, err := LoadSeeds(root)
	if err != nil {
		return summary, []string{fmt.Sprintf("load %s: %v", SeedFileName, err)}
	}

	summary.SeedCount = len(seeds)
	issues := ValidateSeeds(root, seeds)

	repoRoot := filepath.Dir(root)
	for _, seed := range seeds {
		if _, err := resolveCoverPhotoPath(repoRoot, seed.CoverPhoto); err == nil {
			summary.PhotoCount++
		}
	}

	return summary, issues
}

func ValidateSeeds(root string, seeds []SeedRecord) []string {
	var issues []string
	repoRoot := filepath.Dir(root)
	seenURLs := make(map[string]string)

	for _, seed := range seeds {
		ref := seed.reference()

		if !seedIDPattern.MatchString(seed.SeedID) {
			issues = append(issues, fmt.Sprintf("%s: seed_id must match seed-YYYYMMDD-NNN", ref))
		}
		if strings.TrimSpace(seed.XHSProfileURL) == "" {
			issues = append(issues, fmt.Sprintf("%s: xhs_profile_url is required", ref))
		}
		if strings.TrimSpace(seed.CoverPhoto) == "" {
			issues = append(issues, fmt.Sprintf("%s: cover_photo is required", ref))
		}
		if strings.TrimSpace(seed.SourceEvent) == "" {
			issues = append(issues, fmt.Sprintf("%s: source_event is required", ref))
		}
		if _, err := time.Parse(time.RFC3339, seed.CapturedAt); err != nil {
			issues = append(issues, fmt.Sprintf("%s: captured_at must be RFC3339: %v", ref, err))
		}

		normalized, err := NormalizeProfileURL(seed.XHSProfileURL)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: invalid xhs_profile_url: %v", ref, err))
		} else {
			seed.normalizedProfileURL = normalized
			if previous, exists := seenURLs[normalized]; exists {
				issues = append(issues, fmt.Sprintf("%s: normalized profile URL duplicates %s", ref, previous))
			} else {
				seenURLs[normalized] = ref
			}
		}

		if _, err := resolveCoverPhotoPath(repoRoot, seed.CoverPhoto); err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", ref, err))
		}
	}

	return issues
}

func NormalizeProfileURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("scheme must be https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("host is required")
	}

	host := strings.ToLower(parsed.Host)
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	return "https://" + host + path, nil
}

func resolveCoverPhotoPath(repoRoot, coverPhoto string) (string, error) {
	clean := filepath.ToSlash(strings.TrimSpace(coverPhoto))
	if !strings.HasPrefix(clean, "bd/photos/") {
		return "", fmt.Errorf("cover_photo must point into bd/photos/")
	}
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("cover_photo cannot escape bd/photos/")
	}

	abs := filepath.Join(repoRoot, filepath.FromSlash(clean))
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("cover_photo file does not exist: %s", clean)
	}
	return abs, nil
}

func SortSeedsByCapturedAtDesc(seeds []SeedRecord) {
	sort.Slice(seeds, func(i, j int) bool {
		left, _ := time.Parse(time.RFC3339, seeds[i].CapturedAt)
		right, _ := time.Parse(time.RFC3339, seeds[j].CapturedAt)
		return left.After(right)
	})
}

func (s *SeedRecord) applyDefaults() {
	s.SeedID = strings.TrimSpace(s.SeedID)
	s.XHSProfileURL = strings.TrimSpace(s.XHSProfileURL)
	s.CoverPhoto = filepath.ToSlash(strings.TrimSpace(s.CoverPhoto))
	s.SourceEvent = strings.TrimSpace(s.SourceEvent)
	s.CapturedAt = strings.TrimSpace(s.CapturedAt)
	s.DisplayName = strings.TrimSpace(s.DisplayName)
	s.TeamName = strings.TrimSpace(s.TeamName)
	s.TeamScope = strings.TrimSpace(s.TeamScope)
	s.Note = strings.TrimSpace(s.Note)
	s.Collector = strings.TrimSpace(s.Collector)
	if s.Collector == "" {
		s.Collector = DefaultCollector
	}
}

func (s SeedRecord) reference() string {
	if s.SeedID != "" {
		return s.SeedID
	}
	return fmt.Sprintf("line %d", s.lineNumber)
}
