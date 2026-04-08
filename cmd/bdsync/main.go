package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"agentnetwork-red/internal/bd"
)

func main() {
	inboxRoot := flag.String("inbox", bd.InboxDirName, "Path to local inbox root")
	wrapOutput := flag.String("wrap-output", "", "Path to wrap-up HTML output")
	flag.Parse()

	bdRoot := "./bd"
	if flag.NArg() > 0 {
		bdRoot = flag.Arg(0)
	}

	outputPath := *wrapOutput
	if outputPath == "" {
		outputPath = filepath.Join(bdRoot, "display", "index.html")
	}

	summary, err := bd.SyncInbox(bdRoot, *inboxRoot, outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bd sync failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf(
		"bd sync ok: processed=%d created=%d updated=%d needs_review=%d errors=%d skipped=%d wrap=%t\n",
		summary.Processed,
		summary.Created,
		summary.Updated,
		summary.NeedsReview,
		summary.Errors,
		summary.Skipped,
		summary.GeneratedWrap,
	)
}
