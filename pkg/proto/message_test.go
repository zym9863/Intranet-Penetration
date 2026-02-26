package proto

import (
	"bytes"
	"testing"
)

func TestAuthMessageRoundTrip(t *testing.T) {
	msg := &Message{
		Version: 1,
		Type:    MsgAuth,
		Payload: []byte(`{"token":"secret"}`),
	}
	buf := &bytes.Buffer{}
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(buf)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Type != MsgAuth {
		t.Fatalf("expected type %d, got %d", MsgAuth, decoded.Type)
	}
	if string(decoded.Payload) != `{"token":"secret"}` {
		t.Fatalf("payload mismatch: %s", decoded.Payload)
	}
}

func TestNewTunnelMessageRoundTrip(t *testing.T) {
	msg := &Message{
		Version: 1,
		Type:    MsgNewTunnel,
		Payload: []byte(`{"name":"web","type":"tcp","local_port":8080,"remote_port":10080}`),
	}
	buf := &bytes.Buffer{}
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(buf)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Type != MsgNewTunnel {
		t.Fatalf("expected type %d, got %d", MsgNewTunnel, decoded.Type)
	}
}

func TestPingPongRoundTrip(t *testing.T) {
	msg := &Message{Version: 1, Type: MsgPing}
	buf := &bytes.Buffer{}
	if err := msg.Encode(buf); err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(buf)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Type != MsgPing {
		t.Fatalf("expected Ping, got %d", decoded.Type)
	}
	if len(decoded.Payload) != 0 {
		t.Fatal("ping should have empty payload")
	}
}

func TestDecodeInvalidVersion(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.Write([]byte{99, 0x01, 0, 0, 0, 0}) // version=99
	_, err := Decode(buf)
	if err == nil {
		t.Fatal("expected error for invalid version")
	}
}
