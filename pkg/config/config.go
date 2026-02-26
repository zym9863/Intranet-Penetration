package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type TLSConfig struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

type TunnelConfig struct {
	Name           string `yaml:"name"`
	Type           string `yaml:"type"`
	LocalPort      int    `yaml:"local_port"`
	RemotePort     int    `yaml:"remote_port"`
	Domain         string `yaml:"domain"`
	MaxConnections int    `yaml:"max_connections"`
	BandwidthLimit string `yaml:"bandwidth_limit"`
}

type ServerConfig struct {
	BindPort       int       `yaml:"bind_port"`
	Token          string    `yaml:"token"`
	TLS            TLSConfig `yaml:"tls"`
	MaxConnections int       `yaml:"max_connections"`
	HTTPPort       int       `yaml:"http_port"`
	HTTPSPort      int       `yaml:"https_port"`
}

type ClientConfig struct {
	ServerAddr    string         `yaml:"server_addr"`
	Token         string         `yaml:"token"`
	TLSSkipVerify bool           `yaml:"tls_skip_verify"`
	Tunnels       []TunnelConfig `yaml:"tunnels"`
}

func LoadServerConfig(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &ServerConfig{
		BindPort:       7000,
		MaxConnections: 100,
		HTTPPort:       80,
		HTTPSPort:      443,
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadClientConfig(path string) (*ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &ClientConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
