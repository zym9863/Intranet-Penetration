# Chuan - 内网穿透工具

[English](./README-EN.md) | [中文](./README.md)


Chuan (穿) 是一个用 Go 语言实现的高性能内网穿透工具，支持 TCP、UDP、HTTP/HTTPS 协议透传。采用单连接多路复用架构，通过 TLS 加密确保传输安全。

## 特性

- **多协议支持**: TCP、UDP、HTTP/HTTPS 全覆盖
- **高性能**: 基于 smux 单连接多路复用
- **安全可靠**: TLS 1.3 加密 + Token 认证
- **自动重连**: 指数退避算法，断线自动重连
- **灵活配置**: 支持 YAML 配置文件和命令行参数
- **HTTP 反向代理**: 支持域名绑定和泛域名解析

## 架构

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

## 快速开始

### 编译

```bash
go build -o chuan ./cmd/server
go build -o chuan-client ./cmd/client
```

### 服务端部署

1. 创建配置文件 `server.yaml`:

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

2. 启动服务端:

```bash
# 配置文件方式
chuan server -c server.yaml

# 命令行方式
chuan server -p 7000 -t your-secret-token
```

### 客户端配置

1. 创建配置文件 `client.yaml`:

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

2. 启动客户端:

```bash
# 配置文件方式
chuan-client -c client.yaml

# 命令行方式
chuan-client -s your-vps.com:7000 -t your-secret-token \
  --tcp 8080:10080 \
  --udp 53:10053 \
  --http 8080:blog.example.com
```

## 使用场景

### TCP 端口映射

将内网 Web 服务映射到公网:

```bash
# 内网 8080 端口映射到公网 10080
chuan-client -s vps.com:7000 -t token --tcp 8080:10080
```

访问方式: `curl http://vps.com:10080`

### UDP 端口映射

将内网 DNS 服务映射到公网:

```bash
# 内网 53 端口映射到公网 10053
chuan-client -s vps.com:7000 -t token --udp 53:10053
```

### HTTP/HTTPS 反向代理

将内网 Web 服务通过域名暴露:

```bash
# 内网 8080 端口映射到公网域名
chuan-client -s vps.com:7000 -t token --http 8080:blog.example.com
```

## 配置文件说明

### 服务端配置 (server.yaml)

| 参数 | 类型 | 说明 |
|------|------|------|
| bind_port | int | 服务端监听端口，默认 7000 |
| token | string | 认证令牌 |
| tls.cert | string | TLS 证书路径 |
| tls.key | string | TLS 私钥路径 |
| max_connections | int | 最大并发连接数 |
| http_port | int | HTTP 监听端口 |
| https_port | int | HTTPS 监听端口 |

### 客户端配置 (client.yaml)

| 参数 | 类型 | 说明 |
|------|------|------|
| server_addr | string | 服务端地址 (host:port) |
| token | string | 认证令牌 |
| tls_skip_verify | bool | 是否跳过 TLS 证书验证 |
| tunnels | array | 隧道配置列表 |

### 隧道配置 (TunnelConfig)

| 参数 | 类型 | 说明 |
|------|------|------|
| name | string | 隧道名称 |
| type | string | 隧道类型 (tcp/udp/http) |
| local_port | int | 本地服务端口 |
| remote_port | int | 远程映射端口 |
| domain | string | 域名 (HTTP 隧道专用) |

## 通信协议

### 消息格式

```
┌──────────┬──────────┬──────────┬─────────────┐
│ Version  │  Type    │  Length  │   Payload   │
│  1 byte  │  1 byte  │  4 bytes │  N bytes    │
└──────────┴──────────┴──────────┴─────────────┘
```

### 消息类型

| Type | 名称 | 方向 | 说明 |
|------|------|------|------|
| 0x01 | Auth | C→S | 客户端发送 token 认证 |
| 0x02 | AuthResp | S→C | 认证结果 |
| 0x03 | NewTunnel | C→S | 请求注册隧道 |
| 0x04 | NewTunnelResp | S→C | 隧道注册结果 |
| 0x05 | Ping | 双向 | 心跳检测 |
| 0x06 | Pong | 双向 | 心跳响应 |

## 安全特性

- **TLS 加密**: 控制通道强制使用 TLS 1.3
- **Token 认证**: 预共享密钥认证
- **连接限流**: 支持最大连接数和带宽限制
- **心跳检测**: 90 秒无响应自动断开
- **自动重连**: 指数退避算法 (1s → 2s → 4s → ... → 60s)

## 项目结构

```
chuan/
├── cmd/
│   ├── server/
│   │   └── main.go          # 服务端入口
│   └── client/
│       └── main.go          # 客户端入口
├── pkg/
│   ├── proto/
│   │   └── message.go       # 控制协议消息定义与序列化
│   ├── auth/
│   │   └── auth.go          # Token 认证逻辑
│   ├── tunnel/
│   │   ├── tcp.go           # TCP 隧道转发
│   │   ├── udp.go           # UDP 隧道转发
│   │   └── http.go          # HTTP/HTTPS 反向代理
│   ├── mux/
│   │   └── mux.go           # smux 多路复用封装
│   ├── tls/
│   │   └── tls.go           # TLS 配置与证书管理
│   └── config/
│       └── config.go        # 配置文件解析 + 命令行参数合并
├── configs/
│   ├── server.yaml           # 服务端示例配置
│   └── client.yaml          # 客户端示例配置
├── go.mod
└── go.sum
```

## 技术栈

- **Go 1.24+**: 开发语言
- [xtaci/smux](https://github.com/xtaci/smux): 多路复用
- [spf13/cobra](https://github.com/spf13/cobra): CLI 框架
- [yaml.v3](https://gopkg.in/yaml.v3): YAML 配置解析

## 许可证

MIT License - 查看 [LICENSE](./LICENSE) 文件了解详情
