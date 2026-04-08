package skills

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type InstallResult struct {
	Entry      Entry
	TargetRoot string
	TargetDir  string
	SkillPath  string
}

func DefaultInstallRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "skills"), nil
}

func Install(entry Entry, targetRoot string, force bool) (InstallResult, error) {
	if targetRoot == "" {
		defaultRoot, err := DefaultInstallRoot()
		if err != nil {
			return InstallResult{}, err
		}
		targetRoot = defaultRoot
	}

	targetDir := filepath.Join(targetRoot, entry.InstallFolder)
	skillPath := filepath.Join(targetDir, "SKILL.md")
	metadataPath := filepath.Join(targetDir, "metadata.json")

	if _, err := os.Stat(targetDir); err == nil && !force {
		return InstallResult{}, fmt.Errorf("target already exists: %s", targetDir)
	}

	if force {
		if err := os.RemoveAll(targetDir); err != nil {
			return InstallResult{}, err
		}
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return InstallResult{}, err
	}

	if err := copyFile(entry.AbsolutePath, skillPath); err != nil {
		return InstallResult{}, err
	}

	if _, err := os.Stat(entry.MetadataPath); err == nil {
		if err := copyFile(entry.MetadataPath, metadataPath); err != nil {
			return InstallResult{}, err
		}
	}

	return InstallResult{
		Entry:      entry,
		TargetRoot: targetRoot,
		TargetDir:  targetDir,
		SkillPath:  skillPath,
	}, nil
}

func copyFile(source, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer func() {
		_ = dst.Close()
	}()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return dst.Chmod(0o644)
}
