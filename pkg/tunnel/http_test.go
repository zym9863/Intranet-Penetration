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
