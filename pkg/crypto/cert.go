package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// generateCertificate creates a new certificate signed by the provided CA.
// It generates a new ECDSA key pair for the certificate with a random common name.
func generateCertificate(caCertPEM, caKeyPEM []byte) (tls.Certificate, error) {
	var out tls.Certificate

	caKeyDER, _ := pem.Decode(caKeyPEM)
	if caKeyDER == nil {
		return out, fmt.Errorf("failed to decode PEM block from key")
	}

	caKey, err := x509.ParseECPrivateKey(caKeyDER.Bytes)
	if err != nil {
		return out, fmt.Errorf("x509.ParseECPrivateKey(cert): %s", err)
	}

	caCertDER, _ := pem.Decode(caCertPEM)
	if caCertDER == nil {
		return out, fmt.Errorf("failed to decode PEM block from cert: %s", err)
	}
	caCert, err := x509.ParseCertificate(caCertDER.Bytes)
	if err != nil {
		return out, fmt.Errorf("x509.ParseCertificate(cert): %s", err)
	}

	key, err := ecdsa.GenerateKey(caCert.PublicKey.(*ecdsa.PublicKey).Curve, rand.Reader)
	if err != nil {
		return out, fmt.Errorf("failed to generate key pair: %v", err)
	}

	commonName, err := generateRandomString(8, getRandReader(""))
	if err != nil {
		return out, fmt.Errorf("generating random common name: %s", err)
	}

	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC),
		NotAfter:     time.Date(2063, 4, 5, 11, 0, 0, 0, time.UTC),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	cert, err := x509.CreateCertificate(rand.Reader, &tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return out, fmt.Errorf("failed to create client certificate: %v", err)
	}

	out = tls.Certificate{
		Certificate: [][]byte{cert},
		PrivateKey:  key,
	}

	return out, nil
}
