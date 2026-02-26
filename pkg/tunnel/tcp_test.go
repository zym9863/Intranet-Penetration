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
