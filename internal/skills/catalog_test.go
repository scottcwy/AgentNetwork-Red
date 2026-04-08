package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCatalogEnrichesFrontmatter(t *testing.T) {
	repoRoot := t.TempDir()
	skillDir := filepath.Join(repoRoot, "skills", "third_party", "github", "demo", "alpha")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	skillContent := strings.TrimSpace(`
---
name: alpha-skill
description: Does alpha work.
argument-hint: "<alpha>"
---

# Alpha
`)
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "metadata.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	catalogContent := `[
  {
    "id": "github/demo/alpha",
    "provider": "github",
    "local_path": "skills/third_party/github/demo/alpha/SKILL.md",
    "source_repo_url": "https://github.com/demo/alpha",
    "source_path": "SKILL.md",
    "source_raw_url": "https://raw.githubusercontent.com/demo/alpha/main/SKILL.md",
    "discovery_source": "demo",
    "imported_at": "2026-04-08",
    "import_mode": "skill-md-only",
    "status": "unverified"
  }
]`
	catalogPath := filepath.Join(repoRoot, "skills", "catalog.json")
	if err := os.WriteFile(catalogPath, []byte(catalogContent), 0o644); err != nil {
		t.Fatal(err)
	}

	catalog, err := LoadCatalog(catalogPath)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	if len(catalog.Entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(catalog.Entries))
	}
	entry := catalog.Entries[0]
	if entry.SkillName != "alpha-skill" {
		t.Fatalf("expected skill name to be parsed, got %q", entry.SkillName)
	}
	if entry.Description != "Does alpha work." {
		t.Fatalf("expected description to be parsed, got %q", entry.Description)
	}
	if entry.InstallFolder != "github-demo-alpha" {
		t.Fatalf("unexpected install folder: %q", entry.InstallFolder)
	}
}

func TestResolveSupportsSuffixAndSearch(t *testing.T) {
	catalog := &Catalog{
		Entries: []Entry{
			{ID: "github/demo/alpha", SkillName: "alpha-skill", Description: "first"},
			{ID: "github/demo/beta", SkillName: "beta-skill", Description: "second"},
		},
	}

	entry, err := catalog.Resolve("demo/alpha")
	if err != nil {
		t.Fatalf("resolve suffix: %v", err)
	}
	if entry.ID != "github/demo/alpha" {
		t.Fatalf("unexpected entry: %q", entry.ID)
	}

	entry, err = catalog.Resolve("beta-skill")
	if err != nil {
		t.Fatalf("resolve skill name: %v", err)
	}
	if entry.ID != "github/demo/beta" {
		t.Fatalf("unexpected entry: %q", entry.ID)
	}
}

func TestInstallCopiesSkillAndMetadata(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "repo", "skills", "third_party", "github", "demo", "alpha")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sourceSkillPath := filepath.Join(sourceDir, "SKILL.md")
	sourceMetadataPath := filepath.Join(sourceDir, "metadata.json")
	if err := os.WriteFile(sourceSkillPath, []byte("skill body"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceMetadataPath, []byte(`{"id":"github/demo/alpha"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	entry := Entry{
		ID:            "github/demo/alpha",
		AbsolutePath:  sourceSkillPath,
		MetadataPath:  sourceMetadataPath,
		InstallFolder: "github-demo-alpha",
	}

	targetRoot := filepath.Join(root, "install")
	result, err := Install(entry, targetRoot, false)
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	if result.TargetDir != filepath.Join(targetRoot, "github-demo-alpha") {
		t.Fatalf("unexpected target dir: %s", result.TargetDir)
	}
	if _, err := os.Stat(filepath.Join(result.TargetDir, "SKILL.md")); err != nil {
		t.Fatalf("installed SKILL.md missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(result.TargetDir, "metadata.json")); err != nil {
		t.Fatalf("installed metadata.json missing: %v", err)
	}
}
