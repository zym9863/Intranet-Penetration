package mux

import (
	"net"
	"testing"
	"time"
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
		defer stream.Close()
		// Echo back: read a message then write it back
		buf := make([]byte, 1024)
		n, err := stream.Read(buf)
		if err != nil {
			done <- err
			return
		}
		stream.Write(buf[:n])
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
	defer stream.Close()

	msg := []byte("hello chuan")
	stream.Write(msg)

	buf := make([]byte, 1024)
	stream.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := stream.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "hello chuan" {
		t.Fatalf("expected 'hello chuan', got '%s'", string(buf[:n]))
	}
}
