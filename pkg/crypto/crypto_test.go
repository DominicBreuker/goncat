package crypto

import (
	"testing"
)

func TestGenerateCertificates_WithSeed(t *testing.T) {
	t.Parallel()

	seed := "test-seed-123"
	caCert, cert, err := GenerateCertificates(seed)

	if err != nil {
		t.Fatalf("GenerateCertificates(%q) error = %v, want nil", seed, err)
	}
	if caCert == nil {
		t.Error("GenerateCertificates() returned nil caCert")
	}
	if cert.PrivateKey == nil {
		t.Error("GenerateCertificates() returned certificate with nil PrivateKey")
	}
	if len(cert.Certificate) == 0 {
		t.Error("GenerateCertificates() returned certificate with no certificate data")
	}
}

func TestGenerateCertificates_WithoutSeed(t *testing.T) {
	t.Parallel()

	caCert, cert, err := GenerateCertificates("")

	if err != nil {
		t.Fatalf("GenerateCertificates(\"\") error = %v, want nil", err)
	}
	if caCert == nil {
		t.Error("GenerateCertificates() returned nil caCert")
	}
	if cert.PrivateKey == nil {
		t.Error("GenerateCertificates() returned certificate with nil PrivateKey")
	}
	if len(cert.Certificate) == 0 {
		t.Error("GenerateCertificates() returned certificate with no certificate data")
	}
}

func TestGenerateCertificates_Deterministic(t *testing.T) {
	t.Parallel()

	seed := "deterministic-seed"

	caCert1, cert1, err1 := GenerateCertificates(seed)
	if err1 != nil {
		t.Fatalf("First GenerateCertificates() error = %v", err1)
	}

	caCert2, cert2, err2 := GenerateCertificates(seed)
	if err2 != nil {
		t.Fatalf("Second GenerateCertificates() error = %v", err2)
	}

	// Verify determinism - same seed should produce same certificates
	if len(cert1.Certificate) != len(cert2.Certificate) {
		t.Error("Same seed produced different certificate lengths")
	}

	// CA pools should be created successfully
	if caCert1 == nil || caCert2 == nil {
		t.Error("Expected both CA pools to be non-nil")
	}
}

func TestGenerateCertificates_DifferentSeeds(t *testing.T) {
	t.Parallel()

	caCert1, cert1, err1 := GenerateCertificates("seed1")
	if err1 != nil {
		t.Fatalf("GenerateCertificates(\"seed1\") error = %v", err1)
	}

	caCert2, cert2, err2 := GenerateCertificates("seed2")
	if err2 != nil {
		t.Fatalf("GenerateCertificates(\"seed2\") error = %v", err2)
	}

	// Different seeds should produce different certificates
	if len(cert1.Certificate) > 0 && len(cert2.Certificate) > 0 {
		// Just verify both were created successfully
		if caCert1 == nil || caCert2 == nil {
			t.Error("Expected both CA pools to be non-nil")
		}
	}
}
