# Chuan Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go-based intranet tunnel tool (Chuan) that maps local ports to a public server, supporting TCP/UDP/HTTP protocols with TLS encryption, token auth, and rate limiting.

**Architecture:** Client connects to Server over a single TLS-encrypted TCP connection. Communication uses a custom binary control protocol for auth/heartbeat/tunnel negotiation. Data flows through smux-multiplexed virtual streams. Server exposes public-facing ports that proxy traffic back through the tunnel to the client's local services.

**Tech Stack:** Go 1.22+, smux (multiplexing), cobra (CLI), yaml.v3 (config), crypto/tls (encryption), golang.org/x/time/rate (rate limiting)

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/server/main.go`
- Create: `cmd/client/main.go`

**Step 1: Initialize Go module and install dependencies**

Run:
```bash
cd "D:/github/Intranet Penetration"
go mod init github.com/chuan
go get github.com/xtaci/smux/v2
go get github.com/spf13/cobra
go get gopkg.in/yaml.v3
go get golang.org/x/time/rate
```

**Step 2: Create minimal server entry point**

Create `cmd/server/main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("chuan server")
}
```

**Step 3: Create minimal client entry point**

Create `cmd/client/main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("chuan client")
}
```

**Step 4: Verify both build**

Run:
```bash
go build ./cmd/server && go build ./cmd/client
```
Expected: no errors

**Step 5: Commit**

```bash
git add go.mod go.sum cmd/
git commit -m "feat: project scaffolding with Go module and entry points"
```

---

### Task 2: Control Protocol Messages

**Files:**
- Create: `pkg/proto/message.go`
- Create: `pkg/proto/message_test.go`

**Step 1: Write the failing test**

Create `pkg/proto/message_test.go`:
```go
package proto

import (
	"bytes"
	"testing"
)

func TestAuthMessageRoundTrip(t *testing.T) {
	msg := &Message{
		Version: 1,
		Type:    MsgAuth,
		Payload: []byte(`{"token":"secret"}`),
	}
	buf := &bytes.Buffer{}
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(buf)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Type != MsgAuth {
		t.Fatalf("expected type %d, got %d", MsgAuth, decoded.Type)
	}
	if string(decoded.Payload) != `{"token":"secret"}` {
		t.Fatalf("payload mismatch: %s", decoded.Payload)
	}
}

func TestNewTunnelMessageRoundTrip(t *testing.T) {
	msg := &Message{
		Version: 1,
		Type:    MsgNewTunnel,
		Payload: []byte(`{"name":"web","type":"tcp","local_port":8080,"remote_port":10080}`),
	}
	buf := &bytes.Buffer{}
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(buf)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Type != MsgNewTunnel {
		t.Fatalf("expected type %d, got %d", MsgNewTunnel, decoded.Type)
	}
}

func TestPingPongRoundTrip(t *testing.T) {
	msg := &Message{Version: 1, Type: MsgPing}
	buf := &bytes.Buffer{}
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(buf)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Type != MsgPing {
		t.Fatalf("expected Ping, got %d", decoded.Type)
	}
	if len(decoded.Payload) != 0 {
		t.Fatal("ping should have empty payload")
	}
}

func TestDecodeInvalidVersion(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.Write([]byte{99, 0x01, 0, 0, 0, 0}) // version=99
	_, err := Decode(buf)
	if err == nil {
		t.Fatal("expected error for invalid version")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/proto/ -v`
Expected: FAIL (package doesn't exist)

**Step 3: Write implementation**

Create `pkg/proto/message.go`:
```go
package proto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	ProtoVersion = 1
	MaxPayload   = 1 << 20 // 1MB max payload
)

// Message types
const (
	MsgAuth          byte = 0x01
	MsgAuthResp      byte = 0x02
	MsgNewTunnel     byte = 0x03
	MsgNewTunnelResp byte = 0x04
	MsgPing          byte = 0x05
	MsgPong          byte = 0x06
)

// Message is the wire format: [version:1][type:1][length:4][payload:N]
type Message struct {
	Version byte
	Type    byte
	Payload []byte
}

func (m *Message) Encode(w io.Writer) error {
	header := []byte{m.Version, m.Type}
	if _, err := w.Write(header); err != nil {
		return err
	}
	length := uint32(len(m.Payload))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}
	if length > 0 {
		if _, err := w.Write(m.Payload); err != nil {
			return err
		}
	}
	return nil
}

func Decode(r io.Reader) (*Message, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	if header[0] != ProtoVersion {
		return nil, fmt.Errorf("unsupported protocol version: %d", header[0])
	}
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	if length > MaxPayload {
		return nil, errors.New("payload too large")
	}
	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
	}
	return &Message{
		Version: header[0],
		Type:    header[1],
		Payload: payload,
	}, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/proto/ -v`
Expected: PASS (all 4 tests)

**Step 5: Commit**

```bash
git add pkg/proto/
git commit -m "feat: control protocol message encoding/decoding with tests"
```

---

### Task 3: Configuration

**Files:**
- Create: `pkg/config/config.go`
- Create: `pkg/config/config_test.go`
- Create: `configs/server.yaml`
- Create: `configs/client.yaml`

**Step 1: Write the failing test**

Create `pkg/config/config_test.go`:
```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/config/ -v`
Expected: FAIL

**Step 3: Write implementation**

Create `pkg/config/config.go`:
```go
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
```

**Step 4: Run tests**

Run: `go test ./pkg/config/ -v`
Expected: PASS

**Step 5: Create example config files**

Create `configs/server.yaml`:
```yaml
bind_port: 7000
token: "my-secret-token"
tls:
  cert: server.crt
  key: server.key
max_connections: 100
http_port: 80
https_port: 443
```

Create `configs/client.yaml`:
```yaml
server_addr: "your-vps.com:7000"
token: "my-secret-token"
tls_skip_verify: false
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
```

**Step 6: Commit**

```bash
git add pkg/config/ configs/
git commit -m "feat: YAML configuration loading for server and client"
```

---

### Task 4: TLS Helper

**Files:**
- Create: `pkg/tls/tls.go`
- Create: `pkg/tls/tls_test.go`

**Step 1: Write the failing test**

Create `pkg/tls/tls_test.go`:
```go
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"path/filepath"
	"testing"
)

func TestGenerateSelfSignedCert(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "test.crt")
	keyPath := filepath.Join(dir, "test.key")

	err := GenerateSelfSignedCert(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Subject.CommonName != "chuan" {
		t.Fatalf("expected CN=chuan, got %s", parsed.Subject.CommonName)
	}
}

func TestServerTLSConfig(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "s.crt")
	keyPath := filepath.Join(dir, "s.key")
	GenerateSelfSignedCert(certPath, keyPath)

	cfg, err := ServerTLSConfig(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Fatal("expected TLS 1.3 minimum")
	}
}

func TestClientTLSConfig(t *testing.T) {
	cfg := ClientTLSConfig(true)
	if !cfg.InsecureSkipVerify {
		t.Fatal("expected InsecureSkipVerify=true")
	}

	cfg2 := ClientTLSConfig(false)
	if cfg2.InsecureSkipVerify {
		t.Fatal("expected InsecureSkipVerify=false")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/tls/ -v`
Expected: FAIL

**Step 3: Write implementation**

Create `pkg/tls/tls.go`:
```go
package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"time"
)

func GenerateSelfSignedCert(certPath, keyPath string) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "chuan"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}
	certFile, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certFile.Close()
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyFile, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer keyFile.Close()
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}
	pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return nil
}

func ServerTLSConfig(certPath, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

func ClientTLSConfig(skipVerify bool) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: skipVerify,
		MinVersion:         tls.VersionTLS13,
	}
}
```

**Step 4: Run tests**

Run: `go test ./pkg/tls/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/tls/
git commit -m "feat: TLS helper with self-signed cert generation"
```

---

### Task 5: Auth Module

**Files:**
- Create: `pkg/auth/auth.go`
- Create: `pkg/auth/auth_test.go`

**Step 1: Write the failing test**

Create `pkg/auth/auth_test.go`:
```go
package auth

import (
	"bytes"
	"testing"

	"github.com/chuan/pkg/proto"
)

func TestAuthenticateSuccess(t *testing.T) {
	a := NewAuthenticator("my-secret")
	msg := a.BuildAuthMessage()

	buf := &bytes.Buffer{}
	msg.Encode(buf)

	decoded, _ := proto.Decode(buf)
	ok, err := a.Verify(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected auth success")
	}
}

func TestAuthenticateFailure(t *testing.T) {
	server := NewAuthenticator("correct-token")
	client := NewAuthenticator("wrong-token")

	msg := client.BuildAuthMessage()
	buf := &bytes.Buffer{}
	msg.Encode(buf)

	decoded, _ := proto.Decode(buf)
	ok, err := server.Verify(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected auth failure")
	}
}

func TestBuildAuthRespMessage(t *testing.T) {
	resp := BuildAuthRespMessage(true, "ok")
	if resp.Type != proto.MsgAuthResp {
		t.Fatal("wrong type")
	}

	resp2 := BuildAuthRespMessage(false, "bad token")
	if resp2.Type != proto.MsgAuthResp {
		t.Fatal("wrong type")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/auth/ -v`
Expected: FAIL

**Step 3: Write implementation**

Create `pkg/auth/auth.go`:
```go
package auth

import (
	"encoding/json"

	"github.com/chuan/pkg/proto"
)

type authPayload struct {
	Token string `json:"token"`
}

type authRespPayload struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type Authenticator struct {
	token string
}

func NewAuthenticator(token string) *Authenticator {
	return &Authenticator{token: token}
}

func (a *Authenticator) BuildAuthMessage() *proto.Message {
	payload, _ := json.Marshal(authPayload{Token: a.token})
	return &proto.Message{
		Version: proto.ProtoVersion,
		Type:    proto.MsgAuth,
		Payload: payload,
	}
}

func (a *Authenticator) Verify(msg *proto.Message) (bool, error) {
	if msg.Type != proto.MsgAuth {
		return false, nil
	}
	var p authPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return false, err
	}
	return p.Token == a.token, nil
}

func BuildAuthRespMessage(ok bool, message string) *proto.Message {
	payload, _ := json.Marshal(authRespPayload{OK: ok, Message: message})
	return &proto.Message{
		Version: proto.ProtoVersion,
		Type:    proto.MsgAuthResp,
		Payload: payload,
	}
}

func ParseAuthResp(msg *proto.Message) (bool, string, error) {
	var p authRespPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return false, "", err
	}
	return p.OK, p.Message, nil
}
```

**Step 4: Run tests**

Run: `go test ./pkg/auth/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/auth/
git commit -m "feat: token authentication module with tests"
```

---

### Task 6: Mux Wrapper

**Files:**
- Create: `pkg/mux/mux.go`
- Create: `pkg/mux/mux_test.go`

**Step 1: Write the failing test**

Create `pkg/mux/mux_test.go`:
```go
package mux

import (
	"io"
	"net"
	"testing"
)

func TestMuxSessionEcho(t *testing.T) {
	// Create a TCP pipe
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	done := make(chan error, 1)

	// Server side
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			done <- err
			return
		}
		session, err := NewServerSession(conn)
		if err != nil {
			done <- err
			return
		}
		defer session.Close()

		stream, err := session.AcceptStream()
		if err != nil {
			done <- err
			return
		}
		// Echo back
		io.Copy(stream, stream)
		done <- nil
	}()

	// Client side
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewClientSession(conn)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	stream, err := session.OpenStream()
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("hello chuan")
	stream.Write(msg)
	stream.Close()

	buf, err := io.ReadAll(stream)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf) != "hello chuan" {
		t.Fatalf("expected 'hello chuan', got '%s'", string(buf))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/mux/ -v`
Expected: FAIL

**Step 3: Write implementation**

Create `pkg/mux/mux.go`:
```go
package mux

import (
	"net"

	"github.com/xtaci/smux"
)

type Session struct {
	sess *smux.Session
}

func defaultConfig() *smux.Config {
	cfg := smux.DefaultConfig()
	cfg.KeepAliveDisabled = true // We handle heartbeat ourselves
	return cfg
}

func NewServerSession(conn net.Conn) (*Session, error) {
	sess, err := smux.Server(conn, defaultConfig())
	if err != nil {
		return nil, err
	}
	return &Session{sess: sess}, nil
}

func NewClientSession(conn net.Conn) (*Session, error) {
	sess, err := smux.Client(conn, defaultConfig())
	if err != nil {
		return nil, err
	}
	return &Session{sess: sess}, nil
}

func (s *Session) OpenStream() (*smux.Stream, error) {
	return s.sess.OpenStream()
}

func (s *Session) AcceptStream() (*smux.Stream, error) {
	return s.sess.AcceptStream()
}

func (s *Session) Close() error {
	return s.sess.Close()
}

func (s *Session) IsClosed() bool {
	return s.sess.IsClosed()
}

func (s *Session) NumStreams() int {
	return s.sess.NumStreams()
}
```

**Step 4: Run tests**

Run: `go test ./pkg/mux/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/mux/
git commit -m "feat: smux multiplexing wrapper with tests"
```

---

### Task 7: TCP Tunnel

**Files:**
- Create: `pkg/tunnel/tcp.go`
- Create: `pkg/tunnel/tcp_test.go`

**Step 1: Write the failing test**

Create `pkg/tunnel/tcp_test.go`:
```go
package tunnel

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/chuan/pkg/mux"
)

func TestTCPTunnelForward(t *testing.T) {
	// 1. Start a local "backend" TCP server that echoes data
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	go func() {
		for {
			conn, err := backend.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c) // echo
			}(conn)
		}
	}()

	// 2. Create a smux pipe (simulates server<->client connection)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	serverReady := make(chan *mux.Session, 1)
	go func() {
		conn, _ := ln.Accept()
		sess, _ := mux.NewServerSession(conn)
		serverReady <- sess
	}()

	clientConn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	clientSess, err := mux.NewClientSession(clientConn)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSess.Close()
	serverSess := <-serverReady
	defer serverSess.Close()

	// 3. Client side: accept streams and forward to backend
	go func() {
		for {
			stream, err := clientSess.AcceptStream()
			if err != nil {
				return
			}
			go ForwardTCP(stream, backend.Addr().String())
		}
	}()

	// 4. Server side: open a stream (simulates external user connecting)
	stream, err := serverSess.OpenStream()
	if err != nil {
		t.Fatal(err)
	}

	testMsg := []byte("hello tunnel")
	_, err = stream.Write(testMsg)
	if err != nil {
		t.Fatal(err)
	}

	// Read response
	buf := make([]byte, 1024)
	stream.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := stream.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "hello tunnel" {
		t.Fatalf("expected 'hello tunnel', got '%s'", string(buf[:n]))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/tunnel/ -v`
Expected: FAIL

**Step 3: Write implementation**

Create `pkg/tunnel/tcp.go`:
```go
package tunnel

import (
	"io"
	"log"
	"net"
	"sync"
)

// ForwardTCP connects to localAddr and bidirectionally copies data with stream.
func ForwardTCP(stream net.Conn, localAddr string) {
	local, err := net.Dial("tcp", localAddr)
	if err != nil {
		log.Printf("failed to connect to local %s: %v", localAddr, err)
		stream.Close()
		return
	}
	Relay(stream, local)
}

// Relay bidirectionally copies data between two connections, closing both when done.
func Relay(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(a, b)
		a.Close()
	}()
	go func() {
		defer wg.Done()
		io.Copy(b, a)
		b.Close()
	}()
	wg.Wait()
}
```

**Step 4: Run tests**

Run: `go test ./pkg/tunnel/ -v -timeout 10s`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/tunnel/
git commit -m "feat: TCP tunnel forwarding with bidirectional relay"
```

---

### Task 8: UDP Tunnel

**Files:**
- Create: `pkg/tunnel/udp.go`
- Create: `pkg/tunnel/udp_test.go`

**Step 1: Write the failing test**

Create `pkg/tunnel/udp_test.go`:
```go
package tunnel

import (
	"net"
	"testing"
	"time"
)

func TestUDPFrameRoundTrip(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()

	msg := []byte("hello udp frame")

	go func() {
		WriteUDPFrame(a, msg)
	}()

	data, err := ReadUDPFrame(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello udp frame" {
		t.Fatalf("expected 'hello udp frame', got '%s'", string(data))
	}
}

func TestUDPNATTable(t *testing.T) {
	table := NewNATTable(2 * time.Second)
	defer table.Close()

	addr1 := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 1234}
	addr2 := &net.UDPAddr{IP: net.ParseIP("10.0.0.2"), Port: 5678}

	table.Set(addr1.String(), "stream-1")
	table.Set(addr2.String(), "stream-2")

	v, ok := table.Get(addr1.String())
	if !ok || v != "stream-1" {
		t.Fatal("expected stream-1")
	}

	// Wait for expiry
	time.Sleep(3 * time.Second)
	_, ok = table.Get(addr1.String())
	if ok {
		t.Fatal("expected entry to be expired")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/tunnel/ -v -run TestUDP -timeout 15s`
Expected: FAIL

**Step 3: Write implementation**

Create `pkg/tunnel/udp.go`:
```go
package tunnel

import (
	"encoding/binary"
	"io"
	"sync"
	"time"
)

// WriteUDPFrame writes a length-prefixed UDP frame: [2-byte length][data]
func WriteUDPFrame(w io.Writer, data []byte) error {
	header := make([]byte, 2)
	binary.BigEndian.PutUint16(header, uint16(len(data)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// ReadUDPFrame reads a length-prefixed UDP frame.
func ReadUDPFrame(r io.Reader) ([]byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint16(header)
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}

// NATTable maps source addresses to stream identifiers for UDP reply routing.
type NATTable struct {
	mu      sync.RWMutex
	entries map[string]natEntry
	ttl     time.Duration
	done    chan struct{}
}

type natEntry struct {
	value   string
	expires time.Time
}

func NewNATTable(ttl time.Duration) *NATTable {
	t := &NATTable{
		entries: make(map[string]natEntry),
		ttl:     ttl,
		done:    make(chan struct{}),
	}
	go t.cleanup()
	return t
}

func (t *NATTable) Set(key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries[key] = natEntry{value: value, expires: time.Now().Add(t.ttl)}
}

func (t *NATTable) Get(key string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	e, ok := t.entries[key]
	if !ok || time.Now().After(e.expires) {
		return "", false
	}
	return e.value, true
}

func (t *NATTable) cleanup() {
	ticker := time.NewTicker(t.ttl / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.mu.Lock()
			now := time.Now()
			for k, e := range t.entries {
				if now.After(e.expires) {
					delete(t.entries, k)
				}
			}
			t.mu.Unlock()
		case <-t.done:
			return
		}
	}
}

func (t *NATTable) Close() {
	close(t.done)
}
```

**Step 4: Run tests**

Run: `go test ./pkg/tunnel/ -v -run TestUDP -timeout 15s`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/tunnel/udp.go pkg/tunnel/udp_test.go
git commit -m "feat: UDP frame encoding and NAT table with TTL cleanup"
```

---

### Task 9: HTTP Reverse Proxy

**Files:**
- Create: `pkg/tunnel/http.go`
- Create: `pkg/tunnel/http_test.go`

**Step 1: Write the failing test**

Create `pkg/tunnel/http_test.go`:
```go
package tunnel

import (
	"net"
	"testing"

	"github.com/chuan/pkg/mux"
)

func TestHTTPRouterAddAndMatch(t *testing.T) {
	router := NewHTTPRouter()

	// Create a mux pipe
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		mux.NewServerSession(conn)
	}()
	conn, _ := net.Dial("tcp", ln.Addr().String())
	sess, _ := mux.NewClientSession(conn)
	defer sess.Close()

	router.Add("blog.example.com", sess)
	router.Add("api.example.com", sess)

	s, ok := router.Match("blog.example.com")
	if !ok || s != sess {
		t.Fatal("expected to match blog.example.com")
	}

	_, ok = router.Match("unknown.com")
	if ok {
		t.Fatal("should not match unknown domain")
	}

	router.Remove("blog.example.com")
	_, ok = router.Match("blog.example.com")
	if ok {
		t.Fatal("should not match after remove")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/tunnel/ -v -run TestHTTP`
Expected: FAIL

**Step 3: Write implementation**

Create `pkg/tunnel/http.go`:
```go
package tunnel

import (
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/chuan/pkg/mux"
)

// HTTPRouter maps Host headers to mux sessions for HTTP reverse proxy.
type HTTPRouter struct {
	mu     sync.RWMutex
	routes map[string]*mux.Session
}

func NewHTTPRouter() *HTTPRouter {
	return &HTTPRouter{routes: make(map[string]*mux.Session)}
}

func (r *HTTPRouter) Add(domain string, session *mux.Session) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[domain] = session
}

func (r *HTTPRouter) Remove(domain string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.routes, domain)
}

func (r *HTTPRouter) Match(host string) (*mux.Session, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.routes[host]
	return s, ok
}

// HTTPHandler returns an http.Handler that proxies requests through the tunnel.
func (r *HTTPRouter) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		sess, ok := r.Match(req.Host)
		if !ok {
			http.Error(w, "tunnel not found", http.StatusBadGateway)
			return
		}

		stream, err := sess.OpenStream()
		if err != nil {
			http.Error(w, "tunnel unavailable", http.StatusBadGateway)
			return
		}
		defer stream.Close()

		// Hijack the connection to do raw TCP relay
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack not supported", http.StatusInternalServerError)
			return
		}
		clientConn, buf, err := hj.Hijack()
		if err != nil {
			log.Printf("hijack error: %v", err)
			return
		}
		defer clientConn.Close()

		// Write the original request to the stream
		req.Write(stream)

		// If there's buffered data, write it too
		if buf.Reader.Buffered() > 0 {
			buffered := make([]byte, buf.Reader.Buffered())
			buf.Read(buffered)
			stream.Write(buffered)
		}

		// Bidirectional copy
		go func() {
			io.Copy(stream, clientConn)
			stream.Close()
		}()
		io.Copy(clientConn, stream)
	})
}

// ServeTCPHTTP accepts raw TCP connections and proxies HTTP through tunnels.
// This is an alternative to HTTPHandler for cases where you need more control.
func (r *HTTPRouter) ServeTCPHTTP(conn net.Conn, session *mux.Session) {
	stream, err := session.OpenStream()
	if err != nil {
		log.Printf("open stream error: %v", err)
		conn.Close()
		return
	}
	Relay(conn, stream)
}
```

**Step 4: Run tests**

Run: `go test ./pkg/tunnel/ -v -run TestHTTP`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/tunnel/http.go pkg/tunnel/http_test.go
git commit -m "feat: HTTP router and reverse proxy handler"
```

---

### Task 10: Server Core

**Files:**
- Create: `pkg/server/server.go`

This is the central server that ties everything together: TLS listener, auth, tunnel registration, TCP/UDP/HTTP proxy listeners.

**Step 1: Write implementation**

Create `pkg/server/server.go`:
```go
package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/chuan/pkg/auth"
	"github.com/chuan/pkg/config"
	"github.com/chuan/pkg/mux"
	"github.com/chuan/pkg/proto"
	ctls "github.com/chuan/pkg/tls"
	"github.com/chuan/pkg/tunnel"
)

type Server struct {
	cfg         *config.ServerConfig
	auth        *auth.Authenticator
	httpRouter  *tunnel.HTTPRouter
	mu          sync.Mutex
	connections int
	listeners   map[string]net.Listener     // remote_port -> listener
	udpConns    map[string]*net.UDPConn     // remote_port -> udp conn
	natTables   map[string]*tunnel.NATTable // remote_port -> NAT table
}

func New(cfg *config.ServerConfig) *Server {
	return &Server{
		cfg:        cfg,
		auth:       auth.NewAuthenticator(cfg.Token),
		httpRouter: tunnel.NewHTTPRouter(),
		listeners:  make(map[string]net.Listener),
		udpConns:   make(map[string]*net.UDPConn),
		natTables:  make(map[string]*tunnel.NATTable),
	}
}

type tunnelRequest struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	LocalPort  int    `json:"local_port"`
	RemotePort int    `json:"remote_port"`
	Domain     string `json:"domain"`
}

type tunnelResp struct {
	OK         bool   `json:"ok"`
	RemotePort int    `json:"remote_port"`
	Message    string `json:"message"`
}

func (s *Server) Run() error {
	// Load TLS config
	tlsCfg, err := ctls.ServerTLSConfig(s.cfg.TLS.Cert, s.cfg.TLS.Key)
	if err != nil {
		return fmt.Errorf("tls config: %w", err)
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.BindPort))
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Printf("chuan server listening on :%d", s.cfg.BindPort)

	// Start HTTP reverse proxy if configured
	go s.startHTTPProxy()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}

		s.mu.Lock()
		if s.cfg.MaxConnections > 0 && s.connections >= s.cfg.MaxConnections {
			s.mu.Unlock()
			conn.Close()
			log.Printf("max connections reached, rejecting")
			continue
		}
		s.connections++
		s.mu.Unlock()

		go s.handleClient(ctls.UpgradeServerConn(tlsCfg, conn))
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer func() {
		conn.Close()
		s.mu.Lock()
		s.connections--
		s.mu.Unlock()
	}()

	// Read auth message (5 second deadline)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	msg, err := proto.Decode(conn)
	if err != nil {
		log.Printf("read auth: %v", err)
		return
	}
	conn.SetReadDeadline(time.Time{})

	ok, err := s.auth.Verify(msg)
	if err != nil || !ok {
		resp := auth.BuildAuthRespMessage(false, "authentication failed")
		resp.Encode(conn)
		return
	}

	resp := auth.BuildAuthRespMessage(true, "ok")
	resp.Encode(conn)

	// Create mux session
	session, err := mux.NewServerSession(conn)
	if err != nil {
		log.Printf("mux session: %v", err)
		return
	}
	defer session.Close()

	// Handle control stream (first stream for tunnel registration and heartbeat)
	controlStream, err := session.AcceptStream()
	if err != nil {
		log.Printf("accept control stream: %v", err)
		return
	}

	s.handleControl(controlStream, session)
}

func (s *Server) handleControl(control net.Conn, session *mux.Session) {
	defer control.Close()

	for {
		control.SetReadDeadline(time.Now().Add(90 * time.Second))
		msg, err := proto.Decode(control)
		if err != nil {
			log.Printf("control read: %v", err)
			return
		}

		switch msg.Type {
		case proto.MsgPing:
			pong := &proto.Message{Version: proto.ProtoVersion, Type: proto.MsgPong}
			pong.Encode(control)

		case proto.MsgNewTunnel:
			s.handleNewTunnel(msg, control, session)
		}
	}
}

func (s *Server) handleNewTunnel(msg *proto.Message, control net.Conn, session *mux.Session) {
	var req tunnelRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		sendTunnelResp(control, false, 0, "invalid request")
		return
	}

	switch req.Type {
	case "tcp":
		err := s.startTCPProxy(req.RemotePort, session)
		if err != nil {
			sendTunnelResp(control, false, 0, err.Error())
			return
		}
		sendTunnelResp(control, true, req.RemotePort, "ok")
		log.Printf("tunnel [%s] tcp :%d registered", req.Name, req.RemotePort)

	case "udp":
		err := s.startUDPProxy(req.RemotePort, session)
		if err != nil {
			sendTunnelResp(control, false, 0, err.Error())
			return
		}
		sendTunnelResp(control, true, req.RemotePort, "ok")
		log.Printf("tunnel [%s] udp :%d registered", req.Name, req.RemotePort)

	case "http":
		s.httpRouter.Add(req.Domain, session)
		sendTunnelResp(control, true, 0, "ok")
		log.Printf("tunnel [%s] http %s registered", req.Name, req.Domain)

	default:
		sendTunnelResp(control, false, 0, "unknown tunnel type")
	}
}

func (s *Server) startTCPProxy(remotePort int, session *mux.Session) error {
	key := fmt.Sprintf("tcp:%d", remotePort)
	s.mu.Lock()
	if _, exists := s.listeners[key]; exists {
		s.mu.Unlock()
		return fmt.Errorf("port %d already in use", remotePort)
	}
	s.mu.Unlock()

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", remotePort))
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.listeners[key] = ln
	s.mu.Unlock()

	go func() {
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				stream, err := session.OpenStream()
				if err != nil {
					c.Close()
					return
				}
				tunnel.Relay(c, stream)
			}(conn)
		}
	}()
	return nil
}

func (s *Server) startUDPProxy(remotePort int, session *mux.Session) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", remotePort))
	if err != nil {
		return err
	}
	udpConn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("udp:%d", remotePort)
	natTable := tunnel.NewNATTable(60 * time.Second)

	s.mu.Lock()
	s.udpConns[key] = udpConn
	s.natTables[key] = natTable
	s.mu.Unlock()

	go func() {
		defer udpConn.Close()
		defer natTable.Close()
		buf := make([]byte, 65535)
		for {
			n, srcAddr, err := udpConn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			data := make([]byte, n)
			copy(data, buf[:n])

			stream, err := session.OpenStream()
			if err != nil {
				continue
			}

			natTable.Set(srcAddr.String(), srcAddr.String())

			go func(src *net.UDPAddr) {
				defer stream.Close()
				tunnel.WriteUDPFrame(stream, data)

				// Read response
				resp, err := tunnel.ReadUDPFrame(stream)
				if err != nil {
					return
				}
				udpConn.WriteToUDP(resp, src)
			}(srcAddr)
		}
	}()
	return nil
}

func (s *Server) startHTTPProxy() {
	if s.cfg.HTTPPort > 0 {
		log.Printf("HTTP proxy listening on :%d", s.cfg.HTTPPort)
		http.ListenAndServe(fmt.Sprintf(":%d", s.cfg.HTTPPort), s.httpRouter.HTTPHandler())
	}
}

func sendTunnelResp(w net.Conn, ok bool, remotePort int, message string) {
	payload, _ := json.Marshal(tunnelResp{OK: ok, RemotePort: remotePort, Message: message})
	msg := &proto.Message{
		Version: proto.ProtoVersion,
		Type:    proto.MsgNewTunnelResp,
		Payload: payload,
	}
	msg.Encode(w)
}
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/server/`
Expected: no errors

**Step 3: Commit**

```bash
git add pkg/server/
git commit -m "feat: server core with TCP/UDP/HTTP proxy management"
```

---

### Task 11: Client Core

**Files:**
- Create: `pkg/client/client.go`

**Step 1: Write implementation**

Create `pkg/client/client.go`:
```go
package client

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"time"

	"github.com/chuan/pkg/auth"
	"github.com/chuan/pkg/config"
	"github.com/chuan/pkg/mux"
	"github.com/chuan/pkg/proto"
	ctls "github.com/chuan/pkg/tls"
	"github.com/chuan/pkg/tunnel"
)

type Client struct {
	cfg     *config.ClientConfig
	auth    *auth.Authenticator
	session *mux.Session
}

func New(cfg *config.ClientConfig) *Client {
	return &Client{
		cfg:  cfg,
		auth: auth.NewAuthenticator(cfg.Token),
	}
}

type tunnelRequest struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	LocalPort  int    `json:"local_port"`
	RemotePort int    `json:"remote_port"`
	Domain     string `json:"domain"`
}

type tunnelResp struct {
	OK         bool   `json:"ok"`
	RemotePort int    `json:"remote_port"`
	Message    string `json:"message"`
}

func (c *Client) Run() error {
	for {
		err := c.connect()
		if err != nil {
			log.Printf("connection error: %v", err)
		}
		c.reconnect()
	}
}

func (c *Client) connect() error {
	tlsCfg := ctls.ClientTLSConfig(c.cfg.TLSSkipVerify)

	conn, err := net.DialTimeout("tcp", c.cfg.ServerAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	tlsConn := ctls.UpgradeClientConn(tlsCfg, conn, c.cfg.ServerAddr)

	// Authenticate
	authMsg := c.auth.BuildAuthMessage()
	if err := authMsg.Encode(tlsConn); err != nil {
		tlsConn.Close()
		return fmt.Errorf("send auth: %w", err)
	}

	resp, err := proto.Decode(tlsConn)
	if err != nil {
		tlsConn.Close()
		return fmt.Errorf("read auth resp: %w", err)
	}

	ok, msg, err := auth.ParseAuthResp(resp)
	if err != nil || !ok {
		tlsConn.Close()
		return fmt.Errorf("auth failed: %s", msg)
	}
	log.Println("authenticated successfully")

	// Create mux session
	session, err := mux.NewClientSession(tlsConn)
	if err != nil {
		tlsConn.Close()
		return fmt.Errorf("mux session: %w", err)
	}
	c.session = session

	// Open control stream
	controlStream, err := session.OpenStream()
	if err != nil {
		session.Close()
		return fmt.Errorf("control stream: %w", err)
	}

	// Register tunnels
	for _, t := range c.cfg.Tunnels {
		if err := c.registerTunnel(controlStream, t); err != nil {
			log.Printf("register tunnel [%s] failed: %v", t.Name, err)
		}
	}

	// Start accepting data streams (for incoming proxy connections)
	go c.handleStreams(session)

	// Heartbeat loop (blocks until failure)
	return c.heartbeat(controlStream)
}

func (c *Client) registerTunnel(control net.Conn, t config.TunnelConfig) error {
	req := tunnelRequest{
		Name:       t.Name,
		Type:       t.Type,
		LocalPort:  t.LocalPort,
		RemotePort: t.RemotePort,
		Domain:     t.Domain,
	}
	payload, _ := json.Marshal(req)
	msg := &proto.Message{
		Version: proto.ProtoVersion,
		Type:    proto.MsgNewTunnel,
		Payload: payload,
	}
	if err := msg.Encode(control); err != nil {
		return err
	}

	resp, err := proto.Decode(control)
	if err != nil {
		return err
	}
	var tr tunnelResp
	json.Unmarshal(resp.Payload, &tr)
	if !tr.OK {
		return fmt.Errorf("%s", tr.Message)
	}

	log.Printf("tunnel [%s] %s registered (remote port: %d)", t.Name, t.Type, tr.RemotePort)
	return nil
}

func (c *Client) handleStreams(session *mux.Session) {
	for {
		stream, err := session.AcceptStream()
		if err != nil {
			return
		}
		// For TCP/HTTP tunnels, forward to local service
		// The local address is determined by matching the tunnel config
		// For simplicity, each stream carries TCP or HTTP data to the first matching tunnel
		go c.forwardStream(stream)
	}
}

func (c *Client) forwardStream(stream net.Conn) {
	// Try to detect if it's a UDP frame or TCP data
	// For TCP/HTTP: direct relay to local port
	// We use the first TCP tunnel's local port as default
	for _, t := range c.cfg.Tunnels {
		if t.Type == "tcp" || t.Type == "http" {
			tunnel.ForwardTCP(stream, fmt.Sprintf("127.0.0.1:%d", t.LocalPort))
			return
		}
	}
	stream.Close()
}

func (c *Client) heartbeat(control net.Conn) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ping := &proto.Message{Version: proto.ProtoVersion, Type: proto.MsgPing}
		if err := ping.Encode(control); err != nil {
			return fmt.Errorf("ping: %w", err)
		}

		control.SetReadDeadline(time.Now().Add(90 * time.Second))
		resp, err := proto.Decode(control)
		if err != nil {
			return fmt.Errorf("pong: %w", err)
		}
		if resp.Type != proto.MsgPong {
			return fmt.Errorf("expected pong, got %d", resp.Type)
		}
	}
	return nil
}

func (c *Client) reconnect() {
	for attempt := 0; ; attempt++ {
		delay := time.Duration(math.Min(float64(1<<uint(attempt)), 60)) * time.Second
		log.Printf("reconnecting in %v...", delay)
		time.Sleep(delay)
		return // Return to retry connect()
	}
}
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/client/`
Expected: no errors

**Step 3: Commit**

```bash
git add pkg/client/
git commit -m "feat: client core with auth, tunnel registration, heartbeat, and reconnect"
```

---

### Task 12: TLS Upgrade Helpers

We need `UpgradeServerConn` and `UpgradeClientConn` referenced by server/client.

**Files:**
- Modify: `pkg/tls/tls.go`

**Step 1: Add upgrade functions**

Append to `pkg/tls/tls.go`:
```go
func UpgradeServerConn(cfg *tls.Config, conn net.Conn) net.Conn {
	return tls.Server(conn, cfg)
}

func UpgradeClientConn(cfg *tls.Config, conn net.Conn, serverName string) net.Conn {
	cfg.ServerName = serverName
	return tls.Client(conn, cfg)
}
```

Note: the `net` import needs to be added to the existing import block.

**Step 2: Verify all packages build**

Run: `go build ./...`
Expected: no errors

**Step 3: Commit**

```bash
git add pkg/tls/tls.go
git commit -m "feat: TLS connection upgrade helpers"
```

---

### Task 13: CLI Commands (Cobra)

**Files:**
- Rewrite: `cmd/server/main.go`
- Rewrite: `cmd/client/main.go`

**Step 1: Write server CLI**

Rewrite `cmd/server/main.go`:
```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/chuan/pkg/config"
	"github.com/chuan/pkg/server"
	ctls "github.com/chuan/pkg/tls"
	"github.com/spf13/cobra"
)

func main() {
	var configFile string
	var bindPort int
	var token string
	var certFile string
	var keyFile string

	rootCmd := &cobra.Command{
		Use:   "chuan-server",
		Short: "Chuan tunnel server",
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg *config.ServerConfig
			if configFile != "" {
				var err error
				cfg, err = config.LoadServerConfig(configFile)
				if err != nil {
					return err
				}
			} else {
				cfg = &config.ServerConfig{
					BindPort:       bindPort,
					Token:          token,
					MaxConnections: 100,
					HTTPPort:       80,
				}
				cfg.TLS.Cert = certFile
				cfg.TLS.Key = keyFile
			}

			// Override with flags if set
			if cmd.Flags().Changed("port") {
				cfg.BindPort = bindPort
			}
			if cmd.Flags().Changed("token") {
				cfg.Token = token
			}

			if cfg.Token == "" {
				return fmt.Errorf("token is required")
			}

			// Auto-generate cert if not exists
			if cfg.TLS.Cert == "" {
				cfg.TLS.Cert = "chuan-server.crt"
				cfg.TLS.Key = "chuan-server.key"
			}
			if _, err := os.Stat(cfg.TLS.Cert); os.IsNotExist(err) {
				log.Println("generating self-signed certificate...")
				if err := ctls.GenerateSelfSignedCert(cfg.TLS.Cert, cfg.TLS.Key); err != nil {
					return fmt.Errorf("generate cert: %w", err)
				}
			}

			s := server.New(cfg)
			return s.Run()
		},
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file path")
	rootCmd.Flags().IntVarP(&bindPort, "port", "p", 7000, "bind port")
	rootCmd.Flags().StringVarP(&token, "token", "t", "", "auth token")
	rootCmd.Flags().StringVar(&certFile, "cert", "", "TLS cert file")
	rootCmd.Flags().StringVar(&keyFile, "key", "", "TLS key file")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

**Step 2: Write client CLI**

Rewrite `cmd/client/main.go`:
```go
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chuan/pkg/client"
	"github.com/chuan/pkg/config"
	"github.com/spf13/cobra"
)

func main() {
	var configFile string
	var serverAddr string
	var token string
	var tlsSkipVerify bool
	var tcpTunnels []string
	var udpTunnels []string
	var httpTunnels []string

	rootCmd := &cobra.Command{
		Use:   "chuan-client",
		Short: "Chuan tunnel client",
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg *config.ClientConfig
			if configFile != "" {
				var err error
				cfg, err = config.LoadClientConfig(configFile)
				if err != nil {
					return err
				}
			} else {
				cfg = &config.ClientConfig{
					ServerAddr:    serverAddr,
					Token:         token,
					TLSSkipVerify: tlsSkipVerify,
				}
			}

			// Override with flags
			if cmd.Flags().Changed("server") {
				cfg.ServerAddr = serverAddr
			}
			if cmd.Flags().Changed("token") {
				cfg.Token = token
			}
			if cmd.Flags().Changed("tls-skip-verify") {
				cfg.TLSSkipVerify = tlsSkipVerify
			}

			// Parse CLI tunnel flags
			for _, t := range tcpTunnels {
				tc, err := parseTunnelFlag("tcp", t)
				if err != nil {
					return err
				}
				cfg.Tunnels = append(cfg.Tunnels, tc)
			}
			for _, t := range udpTunnels {
				tc, err := parseTunnelFlag("udp", t)
				if err != nil {
					return err
				}
				cfg.Tunnels = append(cfg.Tunnels, tc)
			}
			for _, t := range httpTunnels {
				tc, err := parseHTTPTunnelFlag(t)
				if err != nil {
					return err
				}
				cfg.Tunnels = append(cfg.Tunnels, tc)
			}

			if cfg.ServerAddr == "" {
				return fmt.Errorf("server address is required")
			}
			if cfg.Token == "" {
				return fmt.Errorf("token is required")
			}
			if len(cfg.Tunnels) == 0 {
				return fmt.Errorf("at least one tunnel is required")
			}

			c := client.New(cfg)
			return c.Run()
		},
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file path")
	rootCmd.Flags().StringVarP(&serverAddr, "server", "s", "", "server address (host:port)")
	rootCmd.Flags().StringVarP(&token, "token", "t", "", "auth token")
	rootCmd.Flags().BoolVar(&tlsSkipVerify, "tls-skip-verify", false, "skip TLS verification")
	rootCmd.Flags().StringArrayVar(&tcpTunnels, "tcp", nil, "TCP tunnel (local_port:remote_port)")
	rootCmd.Flags().StringArrayVar(&udpTunnels, "udp", nil, "UDP tunnel (local_port:remote_port)")
	rootCmd.Flags().StringArrayVar(&httpTunnels, "http", nil, "HTTP tunnel (local_port:domain)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// parseTunnelFlag parses "local_port:remote_port" format
func parseTunnelFlag(tunnelType, flag string) (config.TunnelConfig, error) {
	parts := strings.SplitN(flag, ":", 2)
	if len(parts) != 2 {
		return config.TunnelConfig{}, fmt.Errorf("invalid tunnel format: %s (expected local_port:remote_port)", flag)
	}
	local, err := strconv.Atoi(parts[0])
	if err != nil {
		return config.TunnelConfig{}, fmt.Errorf("invalid local port: %s", parts[0])
	}
	remote, err := strconv.Atoi(parts[1])
	if err != nil {
		return config.TunnelConfig{}, fmt.Errorf("invalid remote port: %s", parts[1])
	}
	return config.TunnelConfig{
		Name:       fmt.Sprintf("%s-%d", tunnelType, local),
		Type:       tunnelType,
		LocalPort:  local,
		RemotePort: remote,
	}, nil
}

// parseHTTPTunnelFlag parses "local_port:domain" format
func parseHTTPTunnelFlag(flag string) (config.TunnelConfig, error) {
	parts := strings.SplitN(flag, ":", 2)
	if len(parts) != 2 {
		return config.TunnelConfig{}, fmt.Errorf("invalid http tunnel format: %s (expected local_port:domain)", flag)
	}
	local, err := strconv.Atoi(parts[0])
	if err != nil {
		return config.TunnelConfig{}, fmt.Errorf("invalid local port: %s", parts[0])
	}
	return config.TunnelConfig{
		Name:      fmt.Sprintf("http-%s", parts[1]),
		Type:      "http",
		LocalPort: local,
		Domain:    parts[1],
	}, nil
}
```

**Step 3: Verify build**

Run: `go build ./cmd/server && go build ./cmd/client`
Expected: no errors

**Step 4: Commit**

```bash
git add cmd/
git commit -m "feat: cobra CLI for server and client with config file and flag support"
```

---

### Task 14: Integration Test

**Files:**
- Create: `test/integration_test.go`

**Step 1: Write integration test**

Create `test/integration_test.go`:
```go
package test

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/chuan/pkg/auth"
	"github.com/chuan/pkg/config"
	"github.com/chuan/pkg/mux"
	"github.com/chuan/pkg/proto"
	ctls "github.com/chuan/pkg/tls"
	"github.com/chuan/pkg/tunnel"
)

func TestEndToEndTCPTunnel(t *testing.T) {
	// 1. Generate self-signed cert
	dir := t.TempDir()
	certPath := filepath.Join(dir, "test.crt")
	keyPath := filepath.Join(dir, "test.key")
	ctls.GenerateSelfSignedCert(certPath, keyPath)

	// 2. Start a local echo server (simulates user's local service)
	echoLn, _ := net.Listen("tcp", "127.0.0.1:0")
	defer echoLn.Close()
	go func() {
		for {
			c, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				io.Copy(conn, conn)
			}(c)
		}
	}()

	// 3. Start TLS server (simulates chuan server control port)
	serverTLS, _ := ctls.ServerTLSConfig(certPath, keyPath)
	serverLn, _ := net.Listen("tcp", "127.0.0.1:0")
	defer serverLn.Close()

	token := "test-token-123"
	serverAuth := auth.NewAuthenticator(token)

	serverSessionCh := make(chan *mux.Session, 1)
	go func() {
		raw, _ := serverLn.Accept()
		conn := tls.Server(raw, serverTLS)

		// Read auth
		msg, _ := proto.Decode(conn)
		ok, _ := serverAuth.Verify(msg)
		if !ok {
			t.Error("auth failed on server side")
			return
		}
		resp := auth.BuildAuthRespMessage(true, "ok")
		resp.Encode(conn)

		sess, _ := mux.NewServerSession(conn)
		serverSessionCh <- sess
	}()

	// 4. Client connects
	clientConn, _ := net.Dial("tcp", serverLn.Addr().String())
	clientTLS := ctls.ClientTLSConfig(true)
	tlsConn := tls.Client(clientConn, clientTLS)

	clientAuth := auth.NewAuthenticator(token)
	authMsg := clientAuth.BuildAuthMessage()
	authMsg.Encode(tlsConn)

	authResp, _ := proto.Decode(tlsConn)
	ok, _, _ := auth.ParseAuthResp(authResp)
	if !ok {
		t.Fatal("client auth failed")
	}

	clientSess, _ := mux.NewClientSession(tlsConn)
	defer clientSess.Close()

	serverSess := <-serverSessionCh
	defer serverSess.Close()

	// 5. Client listens for streams and forwards to echo server
	go func() {
		for {
			stream, err := clientSess.AcceptStream()
			if err != nil {
				return
			}
			go tunnel.ForwardTCP(stream, echoLn.Addr().String())
		}
	}()

	// 6. Server opens a stream (simulates external user)
	stream, _ := serverSess.OpenStream()

	testData := "hello end-to-end!"
	fmt.Fprint(stream, testData)

	buf := make([]byte, 1024)
	stream.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := stream.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != testData {
		t.Fatalf("expected '%s', got '%s'", testData, string(buf[:n]))
	}
}
```

**Step 2: Run integration test**

Run: `go test ./test/ -v -timeout 30s`
Expected: PASS

**Step 3: Commit**

```bash
git add test/
git commit -m "test: end-to-end TCP tunnel integration test"
```

---

### Task 15: Run All Tests and Final Verification

**Step 1: Run all tests**

Run: `go test ./... -v -timeout 30s`
Expected: ALL PASS

**Step 2: Build final binaries**

Run:
```bash
go build -o chuan-server ./cmd/server
go build -o chuan-client ./cmd/client
```
Expected: two binaries produced

**Step 3: Test help output**

Run:
```bash
./chuan-server --help
./chuan-client --help
```
Expected: proper cobra help output with all flags

**Step 4: Commit and tag**

```bash
git add -A
git commit -m "chore: final build verification"
git tag v0.1.0
```
