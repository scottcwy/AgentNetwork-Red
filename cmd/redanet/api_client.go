package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agentnetwork-red/internal/config"
)

type apiFlags struct {
	configPath *string
	dataDir    *string
	apiHost    *string
	apiPort    *int
	token      *string
	timeout    *time.Duration
}

type apiClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func addAPIFlags(fs flagSet) *apiFlags {
	return &apiFlags{
		configPath: fs.String("config", "", "Path to config YAML"),
		dataDir:    fs.String("data-dir", "", "Override data directory"),
		apiHost:    fs.String("api-host", "", "Override API host"),
		apiPort:    fs.Int("api-port", 0, "Override API port"),
		token:      fs.String("token", "", "Bearer token override"),
		timeout:    fs.Duration("timeout", 5*time.Second, "Request timeout"),
	}
}

func newAPIClient(flags *apiFlags) (*apiClient, error) {
	cfg, err := config.Load(stringValue(flags.configPath))
	if err != nil {
		return nil, err
	}
	cfg.ApplyCLIOverrides(stringValue(flags.dataDir), stringValue(flags.apiHost), intValue(flags.apiPort))

	token := resolveAPIToken(cfg, stringValue(flags.token))
	return &apiClient{
		baseURL: fmt.Sprintf("http://%s:%d", cfg.APIHost, cfg.APIPort),
		token:   token,
		client:  &http.Client{Timeout: durationValue(flags.timeout)},
	}, nil
}

func (c *apiClient) request(method, path string, query url.Values, body any) ([]byte, int, error) {
	fullURL := c.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var payload []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		payload = encoded
	}

	req, err := http.NewRequest(method, fullURL, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return data, resp.StatusCode, nil
}

func (c *apiClient) get(path string, query url.Values) ([]byte, int, error) {
	return c.request(http.MethodGet, path, query, nil)
}

func (c *apiClient) post(path string, body any) ([]byte, int, error) {
	return c.request(http.MethodPost, path, nil, body)
}

func printJSON(body []byte) error {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, trimmed, "", "  "); err == nil {
		pretty.WriteByte('\n')
		_, err = os.Stdout.Write(pretty.Bytes())
		return err
	}

	if trimmed[len(trimmed)-1] != '\n' {
		trimmed = append(trimmed, '\n')
	}
	_, err := os.Stdout.Write(trimmed)
	return err
}

func ensureHTTPSuccess(status int, body []byte) error {
	if status >= 200 && status < 300 {
		return nil
	}
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(status)
	}
	return fmt.Errorf("api error (%d): %s", status, message)
}

func resolveAPIToken(cfg config.Config, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if env := strings.TrimSpace(os.Getenv("REDANET_API_TOKEN")); env != "" {
		return env
	}
	if env := strings.TrimSpace(os.Getenv("AGENTNETWORK_API_TOKEN")); env != "" {
		return env
	}
	if strings.TrimSpace(cfg.APIToken) != "" {
		return strings.TrimSpace(cfg.APIToken)
	}

	tokenPath := filepath.Join(cfg.DataDir, "api_token")
	data, err := os.ReadFile(tokenPath)
	if err == nil {
		return strings.TrimSpace(string(data))
	}
	return ""
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func durationValue(value *time.Duration) time.Duration {
	if value == nil {
		return 5 * time.Second
	}
	return *value
}

type flagSet interface {
	String(name string, value string, usage string) *string
	Int(name string, value int, usage string) *int
	Duration(name string, value time.Duration, usage string) *time.Duration
}
