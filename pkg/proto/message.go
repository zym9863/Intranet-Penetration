package proto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	ProtoVersion = 1
	MaxPayload   = 1 << 20 // 1MB max payload
)

// Message types
const (
	MsgAuth          byte = 0x01
	MsgAuthResp      byte = 0x02
	MsgNewTunnel     byte = 0x03
	MsgNewTunnelResp byte = 0x04
	MsgPing          byte = 0x05
	MsgPong          byte = 0x06
)

// Message is the wire format: [version:1][type:1][length:4][payload:N]
type Message struct {
	Version byte
	Type    byte
	Payload []byte
}

func (m *Message) Encode(w io.Writer) error {
	header := []byte{m.Version, m.Type}
	if _, err := w.Write(header); err != nil {
		return err
	}
	length := uint32(len(m.Payload))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}
	if length > 0 {
		if _, err := w.Write(m.Payload); err != nil {
			return err
		}
	}
	return nil
}

func Decode(r io.Reader) (*Message, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	if header[0] != ProtoVersion {
		return nil, fmt.Errorf("unsupported protocol version: %d", header[0])
	}
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	if length > MaxPayload {
		return nil, errors.New("payload too large")
	}
	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
	}
	return &Message{
		Version: header[0],
		Type:    header[1],
		Payload: payload,
	}, nil
}
