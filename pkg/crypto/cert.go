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

	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "client"},
		NotBefore:    time.Now().AddDate(-1, 0, 0), // 1 year ago
		NotAfter:     time.Now().AddDate(1, 0, 0),  // 1 year ahead
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
