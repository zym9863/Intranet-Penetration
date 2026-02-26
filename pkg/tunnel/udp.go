package tunnel

import (
	"encoding/binary"
	"io"
	"sync"
	"time"
)

// WriteUDPFrame writes a length-prefixed UDP frame: [2-byte length][data]
func WriteUDPFrame(w io.Writer, data []byte) error {
	header := make([]byte, 2)
	binary.BigEndian.PutUint16(header, uint16(len(data)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// ReadUDPFrame reads a length-prefixed UDP frame.
func ReadUDPFrame(r io.Reader) ([]byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint16(header)
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}

// NATTable maps source addresses to stream identifiers for UDP reply routing.
type NATTable struct {
	mu      sync.RWMutex
	entries map[string]natEntry
	ttl     time.Duration
	done    chan struct{}
}

type natEntry struct {
	value   string
	expires time.Time
}

func NewNATTable(ttl time.Duration) *NATTable {
	t := &NATTable{
		entries: make(map[string]natEntry),
		ttl:     ttl,
		done:    make(chan struct{}),
	}
	go t.cleanup()
	return t
}

func (t *NATTable) Set(key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries[key] = natEntry{value: value, expires: time.Now().Add(t.ttl)}
}

func (t *NATTable) Get(key string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	e, ok := t.entries[key]
	if !ok || time.Now().After(e.expires) {
		return "", false
	}
	return e.value, true
}

func (t *NATTable) cleanup() {
	ticker := time.NewTicker(t.ttl / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.mu.Lock()
			now := time.Now()
			for k, e := range t.entries {
				if now.After(e.expires) {
					delete(t.entries, k)
				}
			}
			t.mu.Unlock()
		case <-t.done:
			return
		}
	}
}

func (t *NATTable) Close() {
	close(t.done)
}
