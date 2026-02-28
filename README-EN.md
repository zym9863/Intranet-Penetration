# Chuan - 内网穿透工具

[English](./README-EN.md) | [中文](./README.md)


Chuan (穿) is a high-performance intranet penetration tool written in Go. It supports TCP, UDP, HTTP/HTTPS protocols and uses a single connection multiplexing architecture with TLS encryption for secure transmission.

## Features

- **Multiple protocol support**: TCP, UDP, HTTP/HTTPS
- **High performance**: built on smux single-connection multiplexing
- **Secure and reliable**: TLS 1.3 encryption + token authentication
- **Auto-reconnect**: exponential backoff algorithm with automatic reconnection
- **Flexible configuration**: YAML config files and command-line options
- **HTTP reverse proxy**: domain binding and wildcard support

## Architecture

```
┌─────────────────┐            ┌─────────────────────────────────┐
│   Chuan Client   │            │         Chuan Server             │
│   (内网机器)      │            │         (公网 VPS)               │
│                  │   TLS +    │                                  │
│  本地服务        │   smux     │   控制通道 (:7000)               │
│  :8080 ◄────┐   │◄──────────►│                                  │
│  :3306 ◄──┐ │   │  单连接     │   TCP代理端口 (:10080, :13306)   │
│  :53   ◄┐ │ │   │  多路复用   │   HTTP反向代理 (:80/:443)        │
│         │ │ │   │            │   UDP代理端口 (:10053)           │
└─────────┼─┼─┼───┘            └──────┼──┼──┼─────────────────────┘
          │ │ │                       │  │  │
          │ │ └── TCP 隧道 ──────────┘  │  │
          │ └──── TCP 隧道 ────────────┘  │
          └────── UDP 隧道 ──────────────┘
```

## Quick Start

### Build

```bash
go build -o chuan ./cmd/server
go build -o chuan-client ./cmd/client
```

### Docker Deployment (Recommended)

1. Start server with Docker Compose:

```bash
docker compose up -d chuan-server
```

2. (Optional) Start server + client together:

```bash
docker compose --profile client up -d
```

3. Check status and logs:

```bash
docker compose ps
docker compose logs -f chuan-server
```

Default config files:

- `configs/server.docker.yaml`
- `configs/client.docker.yaml`

Default exposed ports: `7000`, `80`, `443`, `10080/tcp`, `10053/udp`.
Certificates are auto-generated in the container and persisted to `./data/server/`.

### GitHub Actions Docker Publish (GHCR + Docker Hub)

This repository includes workflow: `.github/workflows/docker-publish.yml`

- Triggers:
  - Push to `main`
  - Push tag `v*` (for example `v1.0.0`)
  - Manual trigger (`workflow_dispatch`)
- Published images:
  - `ghcr.io/<owner>/chuan-server`
  - `ghcr.io/<owner>/chuan-client`
  - `<dockerhub-username>/chuan-server`
  - `<dockerhub-username>/chuan-client`

Set the following in `Settings -> Secrets and variables -> Actions`:

- Repository Variable: `DOCKERHUB_USERNAME` (your Docker Hub username)
- Repository Secret: `DOCKERHUB_TOKEN` (your Docker Hub access token)

GHCR authentication is handled with `GITHUB_TOKEN` automatically.

### Server Deployment

1. Create a config file `server.yaml`:

```yaml
bind_port: 7000
token: "your-secret-token"
tls:
  cert: server.crt
  key: server.key
max_connections: 100
http_port: 80
https_port: 443
```

2. Start the server:

```bash
# using config file
chuan server -c server.yaml

# using command line
chuan server -p 7000 -t your-secret-token
```

### Client Configuration

1. Create `client.yaml`:

```yaml
server_addr: "your-vps.com:7000"
token: "your-secret-token"
tls_skip_verify: false
tunnels:
  - name: web
    type: tcp
    local_port: 8080
    remote_port: 10080
  - name: db
    type: tcp
    local_port: 3306
    remote_port: 13306
  - name: dns
    type: udp
    local_port: 53
    remote_port: 10053
  - name: blog
    type: http
    local_port: 8080
    domain: blog.example.com
```

2. Start the client:

```bash
# config file mode
chuan-client -c client.yaml

# command line mode
chuan-client -s your-vps.com:7000 -t your-secret-token \
  --tcp 8080:10080 \
  --udp 53:10053 \
  --http 8080:blog.example.com
```

## Use Cases

### TCP Port Mapping

Map an internal web service to the public network:

```bash
# internal 8080 mapped to public 10080
chuan-client -s vps.com:7000 -t token --tcp 8080:10080
```

Access via: `curl http://vps.com:10080`

### UDP Port Mapping

Map internal DNS service to public:

```bash
# internal 53 mapped to public 10053
chuan-client -s vps.com:7000 -t token --udp 53:10053
```

### HTTP/HTTPS Reverse Proxy

Expose internal web service through domain:

```bash
# internal 8080 mapped to public domain
chuan-client -s vps.com:7000 -t token --http 8080:blog.example.com
```

## Configuration Files

### Server Config (`server.yaml`)

... (rest of content unchanged) ...
