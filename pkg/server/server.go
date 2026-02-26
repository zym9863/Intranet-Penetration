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
