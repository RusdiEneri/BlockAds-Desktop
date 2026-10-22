package upstream

import (
	"crypto/tls"
	"crypto/x509"
)

// NewTLSConfig creates a standard secure TLS configuration with minimum TLS 1.2,
// server name validation, and strict system root CA checks.
func NewTLSConfig(serverName string, rootCAs *x509.CertPool) *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         serverName,
		RootCAs:            rootCAs,
		InsecureSkipVerify: false,
	}
}
