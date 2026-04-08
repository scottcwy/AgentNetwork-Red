package bd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

func SaveSeeds(root string, seeds []SeedRecord) error {
	SortSeedsByCapturedAtDesc(seeds)

	path := filepath.Join(root, SeedFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data := make([]byte, 0, len(seeds)*160)
	for _, seed := range seeds {
		seed.applyDefaults()
		seed.lineNumber = 0
		seed.normalizedProfileURL = ""
		line, err := json.Marshal(seed)
		if err != nil {
			return err
		}
		data = append(data, line...)
		data = append(data, '\n')
	}

	return os.WriteFile(path, data, 0o644)
}

func EnsureInboxLayout(root string) error {
	for _, dir := range []string{
		filepath.Join(root, DraftsDirName),
		filepath.Join(root, InboxFilesDirName),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
