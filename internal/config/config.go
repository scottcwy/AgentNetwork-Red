package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	defaultAPIPort = 5001
	defaultAPIHost = "127.0.0.1"
)

type Config struct {
	DataDir        string   `yaml:"data_dir"`
	Listen         []string `yaml:"listen"`
	APIHost        string   `yaml:"api_host"`
	APIPort        int      `yaml:"api_port"`
	BootstrapPeers []string `yaml:"bootstrap_peers"`
	APIToken       string   `yaml:"api_token"`
	LogLevel       string   `yaml:"log_level"`
}

func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".redanet"
	}
	return filepath.Join(home, ".redanet")
}

func DefaultConfigPath() string {
	return filepath.Join(DefaultDataDir(), "config.yaml")
}

func Default() Config {
	return Config{
		DataDir: DefaultDataDir(),
		Listen: []string{
			"/ip4/0.0.0.0/tcp/5002",
			"/ip4/0.0.0.0/udp/5002/quic-v1",
		},
		APIHost: defaultAPIHost,
		APIPort: defaultAPIPort,
		BootstrapPeers: []string{
			"/ip4/8.149.141.115/tcp/5002/p2p/12D3KooWDbmAM77BhAUD94g2Vqeik3Bq4ZPKUPnyMGN3oAx6vCH6",
			"/ip4/8.149.141.115/udp/5002/quic-v1/p2p/12D3KooWDbmAM77BhAUD94g2Vqeik3Bq4ZPKUPnyMGN3oAx6vCH6",
		},
		LogLevel: "info",
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = DefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) ApplyCLIOverrides(dataDir, apiHost string, apiPort int) {
	if dataDir != "" {
		c.DataDir = dataDir
	}
	if apiHost != "" {
		c.APIHost = apiHost
	}
	if apiPort > 0 {
		c.APIPort = apiPort
	}
	c.applyDefaults()
}

func (c *Config) SetP2PPort(port int) {
	if port <= 0 {
		return
	}
	c.Listen = []string{
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", port),
	}
}

func (c *Config) ConfigPath(path string) string {
	if path != "" {
		return path
	}
	return DefaultConfigPath()
}

func (c *Config) applyDefaults() {
	if c.DataDir == "" {
		c.DataDir = DefaultDataDir()
	}
	if c.APIHost == "" {
		c.APIHost = defaultAPIHost
	}
	if c.APIPort == 0 {
		c.APIPort = defaultAPIPort
	}
	if len(c.Listen) == 0 {
		c.Listen = Default().Listen
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
}
