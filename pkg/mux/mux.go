package mux

import (
	"net"
	"time"

	"github.com/xtaci/smux/v2"
)

type Session struct {
	sess *smux.Session
}

func defaultConfig() *smux.Config {
	cfg := smux.DefaultConfig()
	cfg.KeepAliveInterval = 365 * 24 * time.Hour // We handle heartbeat ourselves
	cfg.KeepAliveTimeout = 365 * 24 * time.Hour  // Effectively disable smux keepalive
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
