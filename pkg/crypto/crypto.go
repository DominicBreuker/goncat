// Package crypto provides certificate generation and cryptographic utilities
// for establishing secure TLS connections with mutual authentication.
package crypto

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
)

// GenerateCertificates generates a CA certificate pool and a client certificate
// for mutual TLS authentication. If seed is empty, a random seed is generated.
// The same seed will produce the same certificates, enabling key-based authentication.
func GenerateCertificates(seed string) (*x509.CertPool, tls.Certificate, error) {
	var caCert *x509.CertPool
	var cert tls.Certificate
	var err error

	// if seed is unspecified we use a random one
	if seed == "" {
		seed, err = generateRandomString(32, getRandReader(seed))
		if err != nil {
			return caCert, cert, fmt.Errorf("GenerateRandomString(32, getRandReader(seed)): %s", err)
		}
	}

	caKeyPEM, caCertPEM, err := generateKeyPair(seed)
	if err != nil {
		return caCert, cert, fmt.Errorf("generateKeyPair(%s): %s", seed, err)
	}

	caCert = x509.NewCertPool()
	caCert.AppendCertsFromPEM(caCertPEM)

	cert, err = generateCertificate(caCertPEM, caKeyPEM)
	if err != nil {
		return caCert, cert, fmt.Errorf("GenerateClientCert(cert, key): %s", err)
	}

	return caCert, cert, nil
}
