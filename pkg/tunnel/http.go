package tunnel

import (
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/chuan/pkg/mux"
)

// HTTPRouter maps Host headers to mux sessions for HTTP reverse proxy.
type HTTPRouter struct {
	mu     sync.RWMutex
	routes map[string]*mux.Session
}

func NewHTTPRouter() *HTTPRouter {
	return &HTTPRouter{routes: make(map[string]*mux.Session)}
}

func (r *HTTPRouter) Add(domain string, session *mux.Session) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[domain] = session
}

func (r *HTTPRouter) Remove(domain string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.routes, domain)
}

func (r *HTTPRouter) Match(host string) (*mux.Session, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.routes[host]
	return s, ok
}

// HTTPHandler returns an http.Handler that proxies requests through the tunnel.
func (r *HTTPRouter) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		sess, ok := r.Match(req.Host)
		if !ok {
			http.Error(w, "tunnel not found", http.StatusBadGateway)
			return
		}

		stream, err := sess.OpenStream()
		if err != nil {
			http.Error(w, "tunnel unavailable", http.StatusBadGateway)
			return
		}
		defer stream.Close()

		// Hijack the connection to do raw TCP relay
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack not supported", http.StatusInternalServerError)
			return
		}
		clientConn, buf, err := hj.Hijack()
		if err != nil {
			log.Printf("hijack error: %v", err)
			return
		}
		defer clientConn.Close()

		// Write the original request to the stream
		req.Write(stream)

		// If there's buffered data, write it too
		if buf.Reader.Buffered() > 0 {
			buffered := make([]byte, buf.Reader.Buffered())
			buf.Read(buffered)
			stream.Write(buffered)
		}

		// Bidirectional copy
		go func() {
			io.Copy(stream, clientConn)
			stream.Close()
		}()
		io.Copy(clientConn, stream)
	})
}

// ServeTCPHTTP accepts raw TCP connections and proxies HTTP through tunnels.
// This is an alternative to HTTPHandler for cases where you need more control.
func (r *HTTPRouter) ServeTCPHTTP(conn net.Conn, session *mux.Session) {
	stream, err := session.OpenStream()
	if err != nil {
		log.Printf("open stream error: %v", err)
		conn.Close()
		return
	}
	Relay(conn, stream)
}
