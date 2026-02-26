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
