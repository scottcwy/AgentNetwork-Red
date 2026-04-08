package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func runPack(args []string) error {
	fs := flag.NewFlagSet("pack", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) < 1 {
		return fmt.Errorf("usage: %s pack <dir> [out.nut]", os.Args[0])
	}

	sourceDir, err := filepath.Abs(rest[0])
	if err != nil {
		return err
	}
	info, err := os.Stat(sourceDir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("pack source must be a directory")
	}

	outputPath := ""
	if len(rest) > 1 {
		outputPath = rest[1]
	} else {
		outputPath = filepath.Base(sourceDir) + ".nut"
	}
	if filepath.Ext(outputPath) == "" {
		outputPath += ".nut"
	}
	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return err
	}

	if err := writeBundleArchive(sourceDir, outputPath); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "%s\n", outputPath)
	return nil
}

func runUnpack(args []string) error {
	fs := flag.NewFlagSet("unpack", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) < 1 {
		return fmt.Errorf("usage: %s unpack <file.nut> [dir]", os.Args[0])
	}

	bundlePath, err := filepath.Abs(rest[0])
	if err != nil {
		return err
	}

	outputDir := ""
	if len(rest) > 1 {
		outputDir = rest[1]
	} else {
		base := filepath.Base(bundlePath)
		outputDir = strings.TrimSuffix(base, filepath.Ext(base))
		if outputDir == "" {
			outputDir = "bundle"
		}
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		return err
	}

	if err := extractBundleArchive(bundlePath, outputDir); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "%s\n", outputDir)
	return nil
}

func runTaskBundle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s task bundle <upload|get> ...", os.Args[0])
	}

	switch args[0] {
	case "upload":
		return runTaskBundleUpload(args[1:])
	case "get", "download":
		return runTaskBundleGet(args[1:])
	default:
		return fmt.Errorf("unknown task bundle subcommand: %s", args[0])
	}
}

func runTaskBundleUpload(args []string) error {
	fs := flag.NewFlagSet("task bundle upload", flag.ContinueOnError)
	api := addAPIFlags(fs)
	filename := fs.String("filename", "", "Override uploaded bundle filename")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) < 2 {
		return fmt.Errorf("usage: %s task bundle upload [flags] <task-id> <file.nut>", os.Args[0])
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(rest[1])
	if err != nil {
		return err
	}

	name := strings.TrimSpace(*filename)
	if name == "" {
		name = filepath.Base(rest[1])
	}
	query := url.Values{}
	if name != "" {
		query.Set("filename", name)
	}

	body, status, err := client.postBytes(
		path.Join("/api/tasks", url.PathEscape(rest[0]), "bundle"),
		query,
		map[string]string{
			"Content-Type": "application/octet-stream",
			"X-Filename":   name,
		},
		data,
	)
	if err != nil {
		return err
	}
	if err := ensureHTTPSuccess(status, body); err != nil {
		return err
	}
	return printJSON(body)
}

func runTaskBundleGet(args []string) error {
	fs := flag.NewFlagSet("task bundle get", flag.ContinueOnError)
	api := addAPIFlags(fs)
	output := fs.String("out", "", "Write bundle to this file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) < 1 {
		return fmt.Errorf("usage: %s task bundle get [flags] <task-id>", os.Args[0])
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	body, status, err := client.get(path.Join("/api/tasks", url.PathEscape(rest[0]), "bundle"), nil)
	if err != nil {
		return err
	}
	if err := ensureHTTPSuccess(status, body); err != nil {
		return err
	}

	target := strings.TrimSpace(*output)
	if target == "" {
		target = rest[0] + ".nut"
	}
	if err := os.WriteFile(target, body, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "%s\n", target)
	return nil
}

func writeBundleArchive(sourceDir, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	defer archive.Close()

	return filepath.WalkDir(sourceDir, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		relative, err := filepath.Rel(sourceDir, current)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)

		info, err := entry.Info()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relative
		header.Method = zip.Deflate

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		input, err := os.Open(current)
		if err != nil {
			return err
		}
		defer input.Close()

		_, err = io.Copy(writer, input)
		return err
	})
}

func extractBundleArchive(bundlePath, outputDir string) error {
	reader, err := zip.OpenReader(bundlePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	prefix := outputDir + string(os.PathSeparator)
	for _, file := range reader.File {
		targetPath := filepath.Join(outputDir, file.Name)
		cleanTarget := filepath.Clean(targetPath)
		if cleanTarget != outputDir && !strings.HasPrefix(cleanTarget, prefix) {
			return fmt.Errorf("invalid archive path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
			return err
		}

		input, err := file.Open()
		if err != nil {
			return err
		}

		output, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			input.Close()
			return err
		}

		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		inputErr := input.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if inputErr != nil {
			return inputErr
		}
	}

	return nil
}
