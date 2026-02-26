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
		go c.forwardStream(stream)
	}
}

func (c *Client) forwardStream(stream net.Conn) {
	// Forward to the first matching TCP/HTTP tunnel's local port
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
		seconds := math.Min(math.Pow(2, float64(attempt)), 60)
		delay := time.Duration(seconds) * time.Second
		log.Printf("reconnecting in %v...", delay)
		time.Sleep(delay)
		return // Return to retry connect()
	}
}
