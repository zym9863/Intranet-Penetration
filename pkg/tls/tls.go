package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"
)

// GenerateSelfSignedCert creates a self-signed TLS certificate and key pair
// using ECDSA P-256. The certificate has CN=chuan and is valid for 1 year.
func GenerateSelfSignedCert(certPath, keyPath string) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "chuan"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}
	certFile, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certFile.Close()
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyFile, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer keyFile.Close()
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}
	pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return nil
}

// ServerTLSConfig loads a certificate/key pair and returns a *tls.Config
// suitable for a TLS server, enforcing TLS 1.3 as the minimum version.
func ServerTLSConfig(certPath, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// ClientTLSConfig returns a *tls.Config suitable for a TLS client.
// If skipVerify is true, certificate verification is disabled (useful for
// self-signed certs in development). TLS 1.3 is enforced as minimum.
func ClientTLSConfig(skipVerify bool) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: skipVerify,
		MinVersion:         tls.VersionTLS13,
	}
}

// UpgradeServerConn wraps an existing net.Conn in a TLS server connection
// using the provided TLS config.
func UpgradeServerConn(cfg *tls.Config, conn net.Conn) net.Conn {
	return tls.Server(conn, cfg)
}

// UpgradeClientConn wraps an existing net.Conn in a TLS client connection
// using the provided TLS config and server name for SNI.
func UpgradeClientConn(cfg *tls.Config, conn net.Conn, serverName string) net.Conn {
	cfg.ServerName = serverName
	return tls.Client(conn, cfg)
}
