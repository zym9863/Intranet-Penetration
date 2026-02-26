# Chuan - 内网穿透工具设计文档

## 概述

Chuan 是一个用 Go 语言实现的内网穿透工具，核心功能是将内网机器的端口映射到公网。支持 TCP、UDP、HTTP/HTTPS 三种协议类型，采用单连接多路复用架构。

## 整体架构

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

**核心组件**：
- **Server（服务端）**：运行在公网 VPS，监听控制端口，管理隧道，对外暴露代理端口
- **Client（客户端）**：运行在内网机器，主动连接 Server，建立多路复用隧道，将流量转发到本地服务
- **控制通道**：Client 到 Server 的一条 TLS 加密长连接，承载认证、心跳、隧道协商
- **数据通道**：在控制连接上通过 smux 多路复用，每个用户请求对应一个虚拟流

## 通信协议

### 控制协议消息格式

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

### 数据传输

- TCP/HTTP 数据通过 smux 虚拟流直接透传，无额外封装
- UDP 数据帧格式：`[2字节长度][N字节数据]`

### 连接生命周期

```
Client                          Server
  │── TLS Handshake ──────────►│
  │── Auth(token) ─────────────►│
  │◄── AuthResp(ok) ────────────│
  │── NewTunnel(tcp:8080) ─────►│
  │◄── NewTunnelResp(ok:10080) ─│
  │── Ping ────────────────────►│  (每30秒)
  │◄── Pong ────────────────────│
  │                              │
  │  ... 外部用户访问 :10080 ... │
  │◄── smux stream open ────────│
  │── 转发本地:8080数据 ────────►│
```

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
│   └── client.yaml           # 客户端示例配置
├── go.mod
└── go.sum
```

### 核心依赖

- `github.com/xtaci/smux` — 多路复用
- `github.com/spf13/cobra` — 命令行解析
- `gopkg.in/yaml.v3` — YAML 配置解析
- 标准库 `crypto/tls` — TLS 加密

## 配置文件

### server.yaml

```yaml
bind_port: 7000
token: "my-secret-token"
tls:
  cert: server.crt
  key: server.key
max_connections: 100
tunnels_default:
  max_connections: 20
  bandwidth_limit: "10MB"
```

### client.yaml

```yaml
server_addr: "your-vps.com:7000"
token: "my-secret-token"
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

## 安全与限流

### 认证机制

- 客户端连接后第一个消息必须是 Auth，携带预共享 token
- 服务端验证 token，失败则立即断开连接
- token 在配置文件中配置，传输过程通过 TLS 保护

### TLS 加密

- 控制通道默认强制使用 TLS 1.3
- 服务端配置证书和私钥（支持自签名证书）
- 客户端可配置 `tls_skip_verify: true` 跳过证书验证（开发环境用）
- 首次启动时如果没有证书，提供命令自动生成自签名证书

### 连接数与流量限制

- 服务端全局配置 `max_connections`：最大并发连接总数
- 每个隧道可单独配置 `max_connections`：该隧道最大并发数
- 每个隧道可配置 `bandwidth_limit`：带宽限制，通过令牌桶算法实现

### 异常保护

- 心跳超时（90秒无 Pong）自动断开并清理资源
- 客户端断线自动重连，指数退避（1s → 2s → 4s → ... → 60s 上限）
- 服务端优雅关闭：通知所有客户端，等待现有连接排空

## HTTP 反向代理

- 服务端监听 80/443 端口，根据请求的 `Host` 头匹配隧道
- 匹配到隧道后，通过 smux 打开虚拟流，将 HTTP 请求透传到客户端
- HTTPS：服务端做 TLS 终结，配置泛域名证书，内部透传 HTTP

```
用户浏览器                    Chuan Server                  Chuan Client
    │                            │                              │
    │── GET / Host:blog.xx.com ─►│                              │
    │                            │── 查 Host 路由表 ──►匹配隧道  │
    │                            │── smux stream ──────────────►│
    │                            │                    转发到 :8080│
    │◄── HTTP Response ──────────│◄── Response ────────────────│
```

## UDP 隧道处理

- 服务端为 UDP 隧道监听指定 UDP 端口
- 收到 UDP 数据包后，通过 smux 流发送，用 `[2字节长度][数据]` 分帧
- 客户端解帧后将 UDP 包发送到本地目标端口
- 通过记录源地址映射表（`srcAddr → streamID`），保证响应回到正确的请求方
- 映射表条目 60 秒无活动自动清理

## 命令行使用

```bash
# 启动服务端
chuan server -c server.yaml

# 启动客户端（配置文件方式）
chuan client -c client.yaml

# 命令行快捷方式
chuan client -s your-vps.com:7000 -t my-token \
  --tcp 8080:10080 \
  --udp 53:10053 \
  --http 8080:blog.example.com
```
