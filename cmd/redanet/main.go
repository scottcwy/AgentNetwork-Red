package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agentnetwork-red/internal/config"
	"agentnetwork-red/internal/daemon"
	"agentnetwork-red/internal/identity"
	"agentnetwork-red/internal/p2p"
	"agentnetwork-red/internal/store"
)

const version = "0.1.0-dev"

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start":
		if err := runStart(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "status":
		if err := runStatus(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "version":
		fmt.Println(version)
	default:
		usage()
		os.Exit(1)
	}
}

func runStart(args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	configPath := fs.String("config", "", "Path to config YAML")
	dataDir := fs.String("data-dir", "", "Override data directory")
	port := fs.Int("port", 0, "Override libp2p listen port")
	apiHost := fs.String("api-host", "", "Override API host")
	apiPort := fs.Int("api-port", 0, "Override API port")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	cfg.ApplyCLIOverrides(*dataDir, *apiHost, *apiPort)
	cfg.SetP2PPort(*port)

	ident, err := identity.LoadOrCreate(cfg.DataDir)
	if err != nil {
		return err
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := st.Close(); closeErr != nil {
			log.Printf("store close error: %v", closeErr)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("version=%s", version)
	log.Printf("data_dir=%s", cfg.DataDir)
	log.Printf("config=%s", cfg.ConfigPath(*configPath))
	log.Printf("identity_key=%s", identity.KeyPath(cfg.DataDir))
	log.Printf("database=%s", st.Path())
	log.Printf("did=%s", ident.DID)

	if err := st.EnsureInitialBalance(ctx, ident.DID, store.InitialBalance); err != nil {
		return err
	}

	node, err := p2p.New(ctx, cfg, ident)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := node.Close(); closeErr != nil {
			log.Printf("p2p close error: %v", closeErr)
		}
	}()

	log.Printf("peer_id=%s", node.PeerID())
	for _, addr := range node.ListenAddrs() {
		log.Printf("listen_addr=%s", addr)
	}

	d := daemon.New(cfg, ident, st, version).WithP2P(node)
	err = d.Start(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	configPath := fs.String("config", "", "Path to config YAML")
	apiHost := fs.String("api-host", "", "Override API host")
	apiPort := fs.Int("api-port", 0, "Override API port")
	token := fs.String("token", "", "Bearer token override")
	timeout := fs.Duration("timeout", 3*time.Second, "Request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	cfg.ApplyCLIOverrides("", *apiHost, *apiPort)
	if *token != "" {
		cfg.APIToken = *token
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:%d/api/status", cfg.APIHost, cfg.APIPort), nil)
	if err != nil {
		return err
	}
	if cfg.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIToken)
	}

	client := &http.Client{Timeout: *timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if _, err := os.Stdout.Write(body); err != nil {
		return err
	}
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <start|status|version> [flags]\n", os.Args[0])
}
