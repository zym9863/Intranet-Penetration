package tunnel

import (
	"net"
	"testing"
	"time"
)

func TestUDPFrameRoundTrip(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()

	msg := []byte("hello udp frame")

	go func() {
		WriteUDPFrame(a, msg)
	}()

	data, err := ReadUDPFrame(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello udp frame" {
		t.Fatalf("expected 'hello udp frame', got '%s'", string(data))
	}
}

func TestUDPNATTable(t *testing.T) {
	table := NewNATTable(2 * time.Second)
	defer table.Close()

	addr1 := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 1234}
	addr2 := &net.UDPAddr{IP: net.ParseIP("10.0.0.2"), Port: 5678}

	table.Set(addr1.String(), "stream-1")
	table.Set(addr2.String(), "stream-2")

	v, ok := table.Get(addr1.String())
	if !ok || v != "stream-1" {
		t.Fatal("expected stream-1")
	}

	// Wait for expiry
	time.Sleep(3 * time.Second)
	_, ok = table.Get(addr1.String())
	if ok {
		t.Fatal("expected entry to be expired")
	}
}
