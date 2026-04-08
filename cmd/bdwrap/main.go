package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"agentnetwork-red/internal/bd"
)

func main() {
	output := flag.String("output", "", "Output HTML path")
	flag.Parse()

	root := "./bd"
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	outputPath := *output
	if outputPath == "" {
		outputPath = filepath.Join(root, "display", "index.html")
	}

	if err := bd.GenerateWrapUp(root, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "bd wrap failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("bd wrap ok: %s\n", outputPath)
}
