package tls

import (
	"crypto/tls"
	"crypto/x509"
	"path/filepath"
	"testing"
)

func TestGenerateSelfSignedCert(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "test.crt")
	keyPath := filepath.Join(dir, "test.key")

	err := GenerateSelfSignedCert(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Subject.CommonName != "chuan" {
		t.Fatalf("expected CN=chuan, got %s", parsed.Subject.CommonName)
	}
}

func TestServerTLSConfig(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "s.crt")
	keyPath := filepath.Join(dir, "s.key")
	GenerateSelfSignedCert(certPath, keyPath)

	cfg, err := ServerTLSConfig(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Fatal("expected TLS 1.3 minimum")
	}
}

func TestClientTLSConfig(t *testing.T) {
	cfg := ClientTLSConfig(true)
	if !cfg.InsecureSkipVerify {
		t.Fatal("expected InsecureSkipVerify=true")
	}

	cfg2 := ClientTLSConfig(false)
	if cfg2.InsecureSkipVerify {
		t.Fatal("expected InsecureSkipVerify=false")
	}
}
