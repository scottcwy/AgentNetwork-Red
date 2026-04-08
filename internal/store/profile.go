package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var ansNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,62}$`)

type AgentProfile struct {
	DID         string   `json:"did"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Skills      []string `json:"skills,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	PeerID      string   `json:"peer_id,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

type ANSRecord struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	DID          string   `json:"did"`
	Tags         []string `json:"tags,omitempty"`
	RegisteredAt string   `json:"registered_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
}

type AgentDirectoryEntry struct {
	DID          string   `json:"did"`
	Name         string   `json:"name"`
	ProfileName  string   `json:"profile_name,omitempty"`
	Description  string   `json:"description,omitempty"`
	Skills       []string `json:"skills,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	PeerID       string   `json:"peer_id,omitempty"`
	Score        float64  `json:"score"`
	RegisteredAt string   `json:"registered_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
}

func (s *Store) TouchProfilePresence(ctx context.Context, did, peerID string) error {
	now := nowRFC3339()
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO agent_profiles(did, peer_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(did) DO UPDATE SET
		   peer_id=excluded.peer_id,
		   updated_at=excluded.updated_at`,
		did,
		strings.TrimSpace(peerID),
		now,
		now,
	)
	return err
}

func (s *Store) UpsertProfile(ctx context.Context, profile AgentProfile) (AgentProfile, error) {
	now := nowRFC3339()
	profile.DID = strings.TrimSpace(profile.DID)
	if profile.DID == "" {
		return AgentProfile{}, errors.New("missing did")
	}
	profile.Name = strings.TrimSpace(profile.Name)
	profile.Description = strings.TrimSpace(profile.Description)
	profile.Skills = normalizeStringList(profile.Skills)
	profile.Tags = normalizeStringList(profile.Tags)
	profile.PeerID = strings.TrimSpace(profile.PeerID)

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO agent_profiles(did, name, description, skills, tags, peer_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(did) DO UPDATE SET
		   name=excluded.name,
		   description=excluded.description,
		   skills=excluded.skills,
		   tags=excluded.tags,
		   peer_id=excluded.peer_id,
		   updated_at=excluded.updated_at`,
		profile.DID,
		profile.Name,
		profile.Description,
		joinStringList(profile.Skills),
		joinStringList(profile.Tags),
		profile.PeerID,
		now,
		now,
	)
	if err != nil {
		return AgentProfile{}, err
	}
	return s.GetProfile(ctx, profile.DID)
}

func (s *Store) GetProfile(ctx context.Context, did string) (AgentProfile, error) {
	var profile AgentProfile
	var skills, tags string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT did, name, description, skills, tags, peer_id, created_at, updated_at
		 FROM agent_profiles
		 WHERE did = ?`,
		strings.TrimSpace(did),
	).Scan(
		&profile.DID,
		&profile.Name,
		&profile.Description,
		&skills,
		&tags,
		&profile.PeerID,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AgentProfile{}, ErrNotFound
		}
		return AgentProfile{}, err
	}
	profile.Skills = splitStringList(skills)
	profile.Tags = splitStringList(tags)
	return profile, nil
}

func (s *Store) RegisterANS(ctx context.Context, name, did string, tags []string) (ANSRecord, error) {
	normalized, err := normalizeANSName(name)
	if err != nil {
		return ANSRecord{}, err
	}
	did = strings.TrimSpace(did)
	if did == "" {
		return ANSRecord{}, errors.New("missing did")
	}
	tags = normalizeStringList(tags)
	now := nowRFC3339()

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO ans_records(name, did, tags, registered_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET
		   did=excluded.did,
		   tags=excluded.tags,
		   updated_at=excluded.updated_at`,
		normalized,
		did,
		joinStringList(tags),
		now,
		now,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "ans_records.did") || strings.Contains(strings.ToLower(err.Error()), "did") {
			return ANSRecord{}, fmt.Errorf("did already has a registered name: %w", err)
		}
		return ANSRecord{}, err
	}
	return s.ResolveANS(ctx, normalized)
}

func (s *Store) ResolveANS(ctx context.Context, name string) (ANSRecord, error) {
	normalized, err := normalizeANSName(name)
	if err != nil {
		return ANSRecord{}, err
	}

	var record ANSRecord
	var tags string
	err = s.db.QueryRowContext(
		ctx,
		`SELECT name, did, tags, registered_at, updated_at
		 FROM ans_records
		 WHERE name = ?`,
		normalized,
	).Scan(
		&record.DisplayName,
		&record.DID,
		&tags,
		&record.RegisteredAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ANSRecord{}, ErrNotFound
		}
		return ANSRecord{}, err
	}
	record.Name = agentANSDisplay(record.DisplayName)
	record.Tags = splitStringList(tags)
	return record, nil
}

func (s *Store) LookupAgents(ctx context.Context, tags []string, limit int) ([]AgentDirectoryEntry, error) {
	tags = normalizeStringList(tags)
	if len(tags) == 0 {
		return nil, nil
	}
	return s.queryDirectory(ctx, tags, "", limit)
}

func (s *Store) SearchAgents(ctx context.Context, q string, skills []string, limit int) ([]AgentDirectoryEntry, error) {
	return s.queryDirectory(ctx, normalizeStringList(skills), strings.TrimSpace(q), limit)
}

func (s *Store) queryDirectory(ctx context.Context, requiredSkills []string, q string, limit int) ([]AgentDirectoryEntry, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT
		   p.did,
		   p.name,
		   p.description,
		   p.skills,
		   p.tags,
		   p.peer_id,
		   p.updated_at,
		   COALESCE(a.name, ''),
		   COALESCE(a.tags, ''),
		   COALESCE(a.registered_at, '')
		 FROM agent_profiles p
		 LEFT JOIN ans_records a ON a.did = p.did
		 ORDER BY p.updated_at DESC, p.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	queryTerms := tokenizeSearch(q)
	results := make([]AgentDirectoryEntry, 0, limit)
	for rows.Next() {
		var (
			entry         AgentDirectoryEntry
			profileSkills string
			profileTags   string
			ansName       string
			ansTags       string
		)
		if err := rows.Scan(
			&entry.DID,
			&entry.ProfileName,
			&entry.Description,
			&profileSkills,
			&profileTags,
			&entry.PeerID,
			&entry.UpdatedAt,
			&ansName,
			&ansTags,
			&entry.RegisteredAt,
		); err != nil {
			return nil, err
		}

		entry.Skills = splitStringList(profileSkills)
		entry.Tags = mergeStringLists(splitStringList(profileTags), splitStringList(ansTags))
		if ansName != "" {
			entry.Name = agentANSDisplay(ansName)
		} else if entry.ProfileName != "" {
			entry.Name = entry.ProfileName
		} else {
			entry.Name = entry.DID
		}

		if !matchesAllTags(requiredSkills, entry.Skills, entry.Tags) {
			continue
		}
		score := scoreDirectoryEntry(entry, ansName, queryTerms, requiredSkills)
		if len(queryTerms) > 0 && score <= 0 {
			continue
		}
		entry.Score = score
		results = append(results, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	slices.SortFunc(results, func(a, b AgentDirectoryEntry) int {
		switch {
		case a.Score > b.Score:
			return -1
		case a.Score < b.Score:
			return 1
		case a.UpdatedAt > b.UpdatedAt:
			return -1
		case a.UpdatedAt < b.UpdatedAt:
			return 1
		default:
			return strings.Compare(a.DID, b.DID)
		}
	})

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func normalizeANSName(name string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if !ansNamePattern.MatchString(normalized) {
		return "", errors.New("invalid ans name")
	}
	return normalized, nil
}

func normalizeStringList(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, item := range splitStringList(value) {
			normalized := strings.ToLower(strings.TrimSpace(item))
			if normalized == "" {
				continue
			}
			if _, exists := seen[normalized]; exists {
				continue
			}
			seen[normalized] = struct{}{}
			out = append(out, normalized)
		}
	}
	slices.Sort(out)
	return out
}

func splitStringList(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func joinStringList(values []string) string {
	return strings.Join(normalizeStringList(values), ",")
}

func mergeStringLists(groups ...[]string) []string {
	merged := make([]string, 0)
	for _, group := range groups {
		merged = append(merged, group...)
	}
	return normalizeStringList(merged)
}

func matchesAllTags(required []string, groups ...[]string) bool {
	if len(required) == 0 {
		return true
	}
	candidates := mergeStringLists(groups...)
	set := make(map[string]struct{}, len(candidates))
	for _, item := range candidates {
		set[item] = struct{}{}
	}
	for _, item := range required {
		if _, ok := set[item]; !ok {
			return false
		}
	}
	return true
}

func tokenizeSearch(q string) []string {
	return normalizeStringList(strings.Fields(strings.ToLower(strings.TrimSpace(q))))
}

func scoreDirectoryEntry(entry AgentDirectoryEntry, ansName string, queryTerms, requiredSkills []string) float64 {
	score := 0.2 * float64(len(requiredSkills))
	if len(queryTerms) == 0 {
		return score
	}

	joinedSkills := strings.Join(entry.Skills, " ")
	joinedTags := strings.Join(entry.Tags, " ")
	for _, term := range queryTerms {
		switch {
		case strings.Contains(strings.ToLower(ansName), term):
			score += 1.2
		case strings.Contains(strings.ToLower(entry.ProfileName), term):
			score += 1.0
		case strings.Contains(strings.ToLower(entry.Description), term):
			score += 0.8
		case strings.Contains(strings.ToLower(joinedSkills), term):
			score += 0.7
		case strings.Contains(strings.ToLower(joinedTags), term):
			score += 0.6
		case strings.Contains(strings.ToLower(entry.DID), term):
			score += 0.4
		case strings.Contains(strings.ToLower(entry.PeerID), term):
			score += 0.3
		}
	}
	return score
}

func agentANSDisplay(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return "agent://" + name
}
