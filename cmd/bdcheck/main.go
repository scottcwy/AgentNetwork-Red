package main

import (
	"flag"
	"fmt"
	"os"

	"agentnetwork-red/internal/bd"
)

func main() {
	flag.Parse()

	root := "./bd"
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	summary, issues := bd.ValidateRepository(root)
	if len(issues) > 0 {
		fmt.Fprintf(os.Stderr, "bd check failed for %s\n", root)
		for _, issue := range issues {
			fmt.Fprintf(os.Stderr, "- %s\n", issue)
		}
		os.Exit(1)
	}

	fmt.Printf("bd check ok: %d seeds, %d photos\n", summary.SeedCount, summary.PhotoCount)
}
