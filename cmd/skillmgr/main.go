package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agentnetwork-red/internal/skills"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "list":
		if err := runList(os.Args[2:]); err != nil {
			fatal(err)
		}
	case "search":
		if err := runSearch(os.Args[2:]); err != nil {
			fatal(err)
		}
	case "show":
		if err := runShow(os.Args[2:]); err != nil {
			fatal(err)
		}
	case "install":
		if err := runInstall(os.Args[2:]); err != nil {
			fatal(err)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	catalogOverride := fs.String("catalog", "", "Path to catalog.json")
	limit := fs.Int("limit", 0, "Limit number of entries")
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(reorderArgs(args, map[string]bool{
		"-catalog": true,
		"-limit":   true,
		"-json":    false,
	})); err != nil {
		return err
	}

	catalog, err := loadCatalog(*catalogOverride)
	if err != nil {
		return err
	}
	entries := slicesLimit(catalog.Entries, *limit)
	return printEntries(entries, *jsonOutput)
}

func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	catalogOverride := fs.String("catalog", "", "Path to catalog.json")
	limit := fs.Int("limit", 0, "Limit number of entries")
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(reorderArgs(args, map[string]bool{
		"-catalog": true,
		"-limit":   true,
		"-json":    false,
	})); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("search query is required")
	}

	catalog, err := loadCatalog(*catalogOverride)
	if err != nil {
		return err
	}

	query := strings.Join(fs.Args(), " ")
	entries := slicesLimit(catalog.Search(query), *limit)
	return printEntries(entries, *jsonOutput)
}

func runShow(args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	catalogOverride := fs.String("catalog", "", "Path to catalog.json")
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(reorderArgs(args, map[string]bool{
		"-catalog": true,
		"-json":    false,
	})); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("skill query is required")
	}

	catalog, err := loadCatalog(*catalogOverride)
	if err != nil {
		return err
	}
	entry, err := catalog.Resolve(strings.Join(fs.Args(), " "))
	if err != nil {
		return err
	}

	if *jsonOutput {
		return printJSON(entry)
	}

	fmt.Printf("id: %s\n", entry.ID)
	fmt.Printf("name: %s\n", fallbackName(entry))
	fmt.Printf("description: %s\n", fallbackValue(entry.Description, "(none)"))
	fmt.Printf("status: %s\n", fallbackValue(entry.Status, "(none)"))
	fmt.Printf("import_mode: %s\n", fallbackValue(entry.ImportMode, "(none)"))
	fmt.Printf("discovery_source: %s\n", fallbackValue(entry.DiscoverySource, "(none)"))
	fmt.Printf("source_repo: %s\n", entry.SourceRepoURL)
	fmt.Printf("source_path: %s\n", entry.SourcePath)
	fmt.Printf("catalog_path: %s\n", entry.AbsolutePath)
	fmt.Printf("install_folder: %s\n", entry.InstallFolder)
	return nil
}

func runInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	catalogOverride := fs.String("catalog", "", "Path to catalog.json")
	target := fs.String("target", "", "Install root (default: ~/.codex/skills)")
	force := fs.Bool("force", false, "Overwrite existing target directory")
	if err := fs.Parse(reorderArgs(args, map[string]bool{
		"-catalog": true,
		"-target":  true,
		"-force":   false,
	})); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("skill query is required")
	}

	catalog, err := loadCatalog(*catalogOverride)
	if err != nil {
		return err
	}
	entry, err := catalog.Resolve(strings.Join(fs.Args(), " "))
	if err != nil {
		return err
	}

	result, err := skills.Install(entry, *target, *force)
	if err != nil {
		return err
	}

	fmt.Printf("skill install ok: %s -> %s\n", entry.ID, result.TargetDir)
	if entry.Status != "verified" || entry.ImportMode != "" {
		fmt.Printf("warning: installed %s skill (%s)\n", fallbackValue(entry.Status, "unknown-status"), fallbackValue(entry.ImportMode, "unknown-import-mode"))
	}
	return nil
}

func loadCatalog(override string) (*skills.Catalog, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	catalogPath, err := skills.ResolveCatalogPath(cwd, override)
	if err != nil {
		return nil, err
	}
	return skills.LoadCatalog(catalogPath)
}

func printEntries(entries []skills.Entry, jsonOutput bool) error {
	if jsonOutput {
		return printJSON(entries)
	}

	if len(entries) == 0 {
		fmt.Println("no skills found")
		return nil
	}

	for _, entry := range entries {
		fmt.Printf("%s\t%s\t%s\t%s\n", entry.ID, fallbackName(entry), fallbackValue(entry.Status, "-"), entry.SourceRepoURL)
	}
	return nil
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func fallbackName(entry skills.Entry) string {
	if strings.TrimSpace(entry.SkillName) != "" {
		return entry.SkillName
	}
	return filepath.Base(entry.InstallFolder)
}

func fallbackValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func slicesLimit[T any](items []T, limit int) []T {
	if limit <= 0 || limit >= len(items) {
		return items
	}
	return items[:limit]
}

func reorderArgs(args []string, valueFlags map[string]bool) []string {
	var flagArgs []string
	var positional []string

	for index := 0; index < len(args); index++ {
		arg := args[index]
		if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}

		flagArgs = append(flagArgs, arg)
		if strings.Contains(arg, "=") || !valueFlags[arg] {
			continue
		}

		if index+1 < len(args) {
			flagArgs = append(flagArgs, args[index+1])
			index++
		}
	}

	return append(flagArgs, positional...)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "skillmgr failed: %v\n", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <list|search|show|install> [flags]\n", os.Args[0])
}
