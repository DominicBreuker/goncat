package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	mrand "math/rand"
	"time"
)

func generateKeyPair(seed string) ([]byte, []byte, error) {
	key, err := generateCAKey(seed)
	if err != nil {
		return nil, nil, fmt.Errorf("generateKey(%s): %s", seed, err)
	}

	cert, err := generateCACertificate(key, seed)
	if err != nil {
		return nil, nil, fmt.Errorf("generateCertificate(key): %s", err)
	}

	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})

	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal ECDSA private key: %v", err)
	}
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})

	return keyPem, certPem, nil
}

func generateCAKey(seed string) (*ecdsa.PrivateKey, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), getRandReader(seed))
	if err != nil {
		return nil, err
	}

	return priv, nil
}

func generateCACertificate(key *ecdsa.PrivateKey, seed string) ([]byte, error) {
	rng := getRandReader(seed)

	cn, err := generateRandomString(8, rng)
	if err != nil {
		return nil, fmt.Errorf("generating random common name: %s", err)
	}

	org, err := generateRandomString(8, rng)
	if err != nil {
		return nil, fmt.Errorf("generating random organization: %s", err)
	}

	tml := x509.Certificate{
		NotBefore:    time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC),
		NotAfter:     time.Date(2063, 4, 5, 11, 0, 0, 0, time.UTC),
		SerialNumber: big.NewInt(mrand.Int63()),
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{org},
		},
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &tml, &tml, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("creating certificate: %s", err)
	}

	return cert, nil
}
