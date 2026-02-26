package tunnel

import (
	"io"
	"log"
	"net"
	"sync"
)

// ForwardTCP connects to localAddr and bidirectionally copies data with stream.
func ForwardTCP(stream net.Conn, localAddr string) {
	local, err := net.Dial("tcp", localAddr)
	if err != nil {
		log.Printf("failed to connect to local %s: %v", localAddr, err)
		stream.Close()
		return
	}
	Relay(stream, local)
}

// Relay bidirectionally copies data between two connections, closing both when done.
func Relay(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(a, b)
		a.Close()
	}()
	go func() {
		defer wg.Done()
		io.Copy(b, a)
		b.Close()
	}()
	wg.Wait()
}
