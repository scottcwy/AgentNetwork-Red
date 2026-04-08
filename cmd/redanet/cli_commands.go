package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	api := addAPIFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	body, status, err := client.get("/api/status", nil)
	if err != nil {
		return err
	}
	if err := ensureHTTPSuccess(status, body); err != nil {
		return err
	}
	return printJSON(body)
}

func runWhoAmI(args []string) error {
	fs := flag.NewFlagSet("whoami", flag.ContinueOnError)
	api := addAPIFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	body, status, err := client.get("/api/status", nil)
	if err != nil {
		return err
	}
	if err := ensureHTTPSuccess(status, body); err != nil {
		return err
	}

	var payload map[string]any
	if err := jsonUnmarshal(body, &payload); err != nil {
		return printJSON(body)
	}

	output := map[string]any{
		"did":             payload["did"],
		"peer_id":         payload["peer_id"],
		"version":         payload["version"],
		"connected_peers": payload["connected_peers"],
	}
	encoded, err := jsonMarshal(output)
	if err != nil {
		return err
	}
	return printJSON(encoded)
}

func runBalance(args []string) error {
	fs := flag.NewFlagSet("balance", flag.ContinueOnError)
	api := addAPIFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	body, status, err := client.get("/api/credits/balance", nil)
	if err != nil {
		return err
	}
	if err := ensureHTTPSuccess(status, body); err != nil {
		return err
	}
	return printJSON(body)
}

func runPeers(args []string) error {
	fs := flag.NewFlagSet("peers", flag.ContinueOnError)
	api := addAPIFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) == 0 || rest[0] == "list" {
		body, status, err := client.get("/api/peers", nil)
		if err != nil {
			return err
		}
		if err := ensureHTTPSuccess(status, body); err != nil {
			return err
		}
		return printJSON(body)
	}

	if rest[0] == "connect" {
		if len(rest) < 2 {
			return fmt.Errorf("usage: %s peers connect <multiaddr>", os.Args[0])
		}
		body, status, err := client.post("/api/peers/connect", map[string]string{"addr": rest[1]})
		if err != nil {
			return err
		}
		if err := ensureHTTPSuccess(status, body); err != nil {
			return err
		}
		return printJSON(body)
	}

	return fmt.Errorf("unknown peers subcommand: %s", rest[0])
}

func runDiscover(args []string) error {
	fs := flag.NewFlagSet("discover", flag.ContinueOnError)
	api := addAPIFlags(fs)
	limit := fs.Int("limit", 20, "Result limit")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	values := url.Values{}
	if query != "" {
		values.Set("q", query)
	}
	if *limit > 0 {
		values.Set("limit", strconv.Itoa(*limit))
	}

	body, status, err := client.get("/api/discover", values)
	if err != nil {
		return err
	}
	if err := ensureHTTPSuccess(status, body); err != nil {
		return err
	}
	return printJSON(body)
}

func runBoard(args []string) error {
	fs := flag.NewFlagSet("board", flag.ContinueOnError)
	api := addAPIFlags(fs)
	state := fs.String("state", "", "Filter by task state")
	query := fs.String("q", "", "Search query")
	limit := fs.Int("limit", 50, "Result limit")
	stats := fs.Bool("stats", false, "Show board stats instead of tasks")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	endpoint := "/api/tasks/board"
	values := url.Values{}
	if *stats {
		endpoint = "/api/tasks/board/stats"
	} else {
		if strings.TrimSpace(*state) != "" {
			values.Set("state", strings.TrimSpace(*state))
		}
		if strings.TrimSpace(*query) != "" {
			values.Set("q", strings.TrimSpace(*query))
		}
		if *limit > 0 {
			values.Set("limit", strconv.Itoa(*limit))
		}
	}

	body, status, err := client.get(endpoint, values)
	if err != nil {
		return err
	}
	if err := ensureHTTPSuccess(status, body); err != nil {
		return err
	}
	return printJSON(body)
}

func runTask(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s task <publish|get|list|stats|claim|submit|accept|reject|cancel|abandon|dispute|arbitrate>", os.Args[0])
	}

	switch args[0] {
	case "publish":
		return runTaskPublish(args[1:])
	case "get":
		return runTaskGet(args[1:])
	case "list", "board":
		return runBoard(args[1:])
	case "stats":
		return runBoard(append(args[1:], "-stats"))
	case "claim", "accept", "reject", "cancel", "abandon", "dispute":
		return runTaskAction(args[0], args[1:], nil)
	case "submit":
		return runTaskSubmit(args[1:])
	case "arbitrate":
		return runTaskArbitrate(args[1:])
	default:
		return fmt.Errorf("unknown task subcommand: %s", args[0])
	}
}

func runTaskPublish(args []string) error {
	fs := flag.NewFlagSet("task publish", flag.ContinueOnError)
	api := addAPIFlags(fs)
	tags := fs.String("tags", "", "Comma-separated tags")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) < 2 {
		return fmt.Errorf("usage: %s task publish [flags] <title> <reward> [description]", os.Args[0])
	}

	reward, err := strconv.ParseInt(rest[1], 10, 64)
	if err != nil {
		return fmt.Errorf("parse reward: %w", err)
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	body, status, err := client.post("/api/tasks", map[string]any{
		"title":       rest[0],
		"reward":      reward,
		"description": strings.Join(rest[2:], " "),
		"tags":        strings.TrimSpace(*tags),
	})
	if err != nil {
		return err
	}
	if err := ensureHTTPSuccess(status, body); err != nil {
		return err
	}
	return printJSON(body)
}

func runTaskGet(args []string) error {
	fs := flag.NewFlagSet("task get", flag.ContinueOnError)
	api := addAPIFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) < 1 {
		return fmt.Errorf("usage: %s task get <task-id>", os.Args[0])
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	body, status, err := client.get("/api/tasks/"+url.PathEscape(fs.Args()[0]), nil)
	if err != nil {
		return err
	}
	if err := ensureHTTPSuccess(status, body); err != nil {
		return err
	}
	return printJSON(body)
}

func runTaskAction(action string, args []string, payload any) error {
	fs := flag.NewFlagSet("task "+action, flag.ContinueOnError)
	api := addAPIFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) < 1 {
		return fmt.Errorf("usage: %s task %s <task-id>", os.Args[0], action)
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	body, status, err := client.post(path.Join("/api/tasks", url.PathEscape(fs.Args()[0]), action), payload)
	if err != nil {
		return err
	}
	if err := ensureHTTPSuccess(status, body); err != nil {
		return err
	}
	return printJSON(body)
}

func runTaskSubmit(args []string) error {
	fs := flag.NewFlagSet("task submit", flag.ContinueOnError)
	api := addAPIFlags(fs)
	result := fs.String("result", "", "Submission result text")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) < 1 {
		return fmt.Errorf("usage: %s task submit [flags] <task-id> [result]", os.Args[0])
	}

	value := strings.TrimSpace(*result)
	if value == "" && len(rest) > 1 {
		value = strings.Join(rest[1:], " ")
	}
	if value == "" {
		return fmt.Errorf("submit result cannot be empty")
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	body, status, err := client.post(path.Join("/api/tasks", url.PathEscape(rest[0]), "submit"), map[string]string{"result": value})
	if err != nil {
		return err
	}
	if err := ensureHTTPSuccess(status, body); err != nil {
		return err
	}
	return printJSON(body)
}

func runTaskArbitrate(args []string) error {
	fs := flag.NewFlagSet("task arbitrate", flag.ContinueOnError)
	api := addAPIFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) < 2 {
		return fmt.Errorf("usage: %s task arbitrate <task-id> <favor_claimant|favor_publisher>", os.Args[0])
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	body, status, err := client.post(path.Join("/api/tasks", url.PathEscape(fs.Args()[0]), "arbitrate"), map[string]string{"verdict": fs.Args()[1]})
	if err != nil {
		return err
	}
	if err := ensureHTTPSuccess(status, body); err != nil {
		return err
	}
	return printJSON(body)
}

func runChat(args []string) error {
	fs := flag.NewFlagSet("chat", flag.ContinueOnError)
	api := addAPIFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := newAPIClient(api)
	if err != nil {
		return err
	}

	rest := fs.Args()
	switch {
	case len(rest) == 0:
		body, status, err := client.get("/api/dm/inbox", nil)
		if err != nil {
			return err
		}
		if err := ensureHTTPSuccess(status, body); err != nil {
			return err
		}
		return printJSON(body)
	case rest[0] == "inbox":
		body, status, err := client.get("/api/dm/inbox", nil)
		if err != nil {
			return err
		}
		if err := ensureHTTPSuccess(status, body); err != nil {
			return err
		}
		return printJSON(body)
	case rest[0] == "thread" && len(rest) >= 2:
		body, status, err := client.get("/api/dm/thread/"+url.PathEscape(rest[1]), nil)
		if err != nil {
			return err
		}
		if err := ensureHTTPSuccess(status, body); err != nil {
			return err
		}
		return printJSON(body)
	case len(rest) == 1:
		body, status, err := client.get("/api/dm/thread/"+url.PathEscape(rest[0]), nil)
		if err != nil {
			return err
		}
		if err := ensureHTTPSuccess(status, body); err != nil {
			return err
		}
		return printJSON(body)
	default:
		body, status, err := client.post("/api/dm/send", map[string]string{
			"to":        rest[0],
			"plaintext": strings.Join(rest[1:], " "),
		})
		if err != nil {
			return err
		}
		if err := ensureHTTPSuccess(status, body); err != nil {
			return err
		}
		return printJSON(body)
	}
}

func jsonMarshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func jsonUnmarshal(data []byte, target any) error {
	return json.Unmarshal(data, target)
}
