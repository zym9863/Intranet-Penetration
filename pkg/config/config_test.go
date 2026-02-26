package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadServerConfig(t *testing.T) {
	content := `
bind_port: 7000
token: "test-token"
tls:
  cert: server.crt
  key: server.key
max_connections: 50
`
	path := filepath.Join(t.TempDir(), "server.yaml")
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := LoadServerConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BindPort != 7000 {
		t.Fatalf("expected port 7000, got %d", cfg.BindPort)
	}
	if cfg.Token != "test-token" {
		t.Fatalf("expected token test-token, got %s", cfg.Token)
	}
	if cfg.MaxConnections != 50 {
		t.Fatalf("expected 50, got %d", cfg.MaxConnections)
	}
}

func TestLoadClientConfig(t *testing.T) {
	content := `
server_addr: "example.com:7000"
token: "my-token"
tls_skip_verify: true
tunnels:
  - name: web
    type: tcp
    local_port: 8080
    remote_port: 10080
  - name: dns
    type: udp
    local_port: 53
    remote_port: 10053
  - name: blog
    type: http
    local_port: 8080
    domain: blog.example.com
`
	path := filepath.Join(t.TempDir(), "client.yaml")
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := LoadClientConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerAddr != "example.com:7000" {
		t.Fatalf("expected example.com:7000, got %s", cfg.ServerAddr)
	}
	if !cfg.TLSSkipVerify {
		t.Fatal("expected tls_skip_verify=true")
	}
	if len(cfg.Tunnels) != 3 {
		t.Fatalf("expected 3 tunnels, got %d", len(cfg.Tunnels))
	}
	if cfg.Tunnels[2].Type != "http" || cfg.Tunnels[2].Domain != "blog.example.com" {
		t.Fatal("http tunnel mismatch")
	}
}

func TestServerConfigDefaults(t *testing.T) {
	content := `token: "t"`
	path := filepath.Join(t.TempDir(), "s.yaml")
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := LoadServerConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BindPort != 7000 {
		t.Fatalf("default port should be 7000, got %d", cfg.BindPort)
	}
	if cfg.MaxConnections != 100 {
		t.Fatalf("default max_connections should be 100, got %d", cfg.MaxConnections)
	}
}
