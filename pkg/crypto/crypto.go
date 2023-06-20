package crypto

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
)

// GenerateCertificates ...
func GenerateCertificates(seed string) (*x509.CertPool, tls.Certificate, error) {
	var caCert *x509.CertPool
	var cert tls.Certificate
	var err error

	// if seed is unspecified we use a random one
	if seed == "" {
		seed, err = GenerateRandomString(32)
		if err != nil {
			return caCert, cert, fmt.Errorf("GenerateRandomString(32): %s", err)
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
