package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultCatalogRelativePath = "skills/catalog.json"

var errCatalogNotFound = errors.New("skills catalog not found")

type Entry struct {
	ID              string `json:"id"`
	Provider        string `json:"provider"`
	LocalPath       string `json:"local_path"`
	SourceRepoURL   string `json:"source_repo_url"`
	SourcePath      string `json:"source_path"`
	SourceRawURL    string `json:"source_raw_url"`
	DiscoverySource string `json:"discovery_source"`
	ImportedAt      string `json:"imported_at"`
	ImportMode      string `json:"import_mode"`
	Status          string `json:"status"`

	SkillName     string `json:"-"`
	Description   string `json:"-"`
	ArgumentHint  string `json:"-"`
	AbsolutePath  string `json:"-"`
	MetadataPath  string `json:"-"`
	InstallFolder string `json:"-"`
}

type Catalog struct {
	RepoRoot string
	Path     string
	Entries  []Entry
}

type skillFrontmatter struct {
	Name         string `yaml:"name"`
	Description  string `yaml:"description"`
	ArgumentHint string `yaml:"argument-hint"`
}

func ResolveCatalogPath(startDir, override string) (string, error) {
	if override != "" {
		abs, err := filepath.Abs(override)
		if err != nil {
			return "", err
		}
		return abs, nil
	}

	current, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(current, defaultCatalogRelativePath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", errCatalogNotFound
		}
		current = parent
	}
}

func LoadCatalog(catalogPath string) (*Catalog, error) {
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode catalog: %w", err)
	}

	repoRoot := filepath.Dir(filepath.Dir(catalogPath))
	for index := range entries {
		entry := &entries[index]
		entry.AbsolutePath = filepath.Join(repoRoot, filepath.FromSlash(entry.LocalPath))
		entry.MetadataPath = filepath.Join(filepath.Dir(entry.AbsolutePath), "metadata.json")
		entry.InstallFolder = makeInstallFolderName(entry.ID)

		frontmatter, err := readFrontmatter(entry.AbsolutePath)
		if err != nil {
			return nil, fmt.Errorf("read frontmatter for %s: %w", entry.ID, err)
		}
		entry.SkillName = frontmatter.Name
		entry.Description = frontmatter.Description
		entry.ArgumentHint = frontmatter.ArgumentHint
	}

	slices.SortFunc(entries, func(left, right Entry) int {
		return strings.Compare(left.ID, right.ID)
	})

	return &Catalog{
		RepoRoot: repoRoot,
		Path:     catalogPath,
		Entries:  entries,
	}, nil
}

func (c *Catalog) Search(query string) []Entry {
	if strings.TrimSpace(query) == "" {
		return slices.Clone(c.Entries)
	}

	needle := strings.ToLower(strings.TrimSpace(query))
	var matches []Entry
	for _, entry := range c.Entries {
		if strings.Contains(strings.ToLower(searchText(entry)), needle) {
			matches = append(matches, entry)
		}
	}
	return matches
}

func (c *Catalog) Resolve(query string) (Entry, error) {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return Entry{}, errors.New("query is required")
	}

	if entry, ok := c.findOne(func(entry Entry) bool {
		return strings.EqualFold(entry.ID, needle)
	}); ok {
		return entry, nil
	}

	if entry, ok := c.findOne(func(entry Entry) bool {
		return strings.EqualFold(strings.TrimPrefix(entry.ID, entry.Provider+"/"), needle)
	}); ok {
		return entry, nil
	}

	if entry, ok := c.findOne(func(entry Entry) bool {
		return strings.EqualFold(entry.InstallFolder, needle)
	}); ok {
		return entry, nil
	}

	if entry, ok := c.findOne(func(entry Entry) bool {
		return strings.EqualFold(filepath.Base(entry.AbsolutePath), needle) ||
			strings.EqualFold(filepath.Base(filepath.Dir(entry.AbsolutePath)), needle) ||
			strings.EqualFold(entry.SkillName, needle)
	}); ok {
		return entry, nil
	}

	matches := c.Search(query)
	switch len(matches) {
	case 0:
		return Entry{}, fmt.Errorf("no skill matches %q", query)
	case 1:
		return matches[0], nil
	default:
		var ids []string
		for _, match := range matches {
			ids = append(ids, match.ID)
		}
		return Entry{}, fmt.Errorf("query %q is ambiguous: %s", query, strings.Join(ids, ", "))
	}
}

func readFrontmatter(path string) (skillFrontmatter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return skillFrontmatter{}, err
	}

	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return skillFrontmatter{}, nil
	}

	rest := strings.TrimPrefix(content, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end == -1 {
		return skillFrontmatter{}, nil
	}

	var meta skillFrontmatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &meta); err != nil {
		return skillFrontmatter{}, err
	}
	return meta, nil
}

func searchText(entry Entry) string {
	return strings.Join([]string{
		entry.ID,
		entry.Provider,
		entry.SkillName,
		entry.Description,
		entry.ArgumentHint,
		entry.SourceRepoURL,
		entry.SourcePath,
		entry.DiscoverySource,
		entry.ImportMode,
		entry.Status,
	}, "\n")
}

func makeInstallFolderName(id string) string {
	replacer := strings.NewReplacer("/", "-", "_", "-", ".", "-")
	name := strings.ToLower(replacer.Replace(id))
	name = strings.Trim(name, "-")
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	return name
}

func (c *Catalog) findOne(match func(Entry) bool) (Entry, bool) {
	var found []Entry
	for _, entry := range c.Entries {
		if match(entry) {
			found = append(found, entry)
		}
	}
	if len(found) == 1 {
		return found[0], true
	}
	return Entry{}, false
}
