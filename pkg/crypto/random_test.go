package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestGetRandReader_WithSeed(t *testing.T) {
	t.Parallel()

	seed := "test-seed"
	r := getRandReader(seed)

	if r == nil {
		t.Error("getRandReader() returned nil")
	}

	// Verify it returns a deterministic reader
	buf1 := make([]byte, 32)
	buf2 := make([]byte, 32)

	r1 := getRandReader(seed)
	r2 := getRandReader(seed)

	if _, err := r1.Read(buf1); err != nil {
		t.Fatalf("First read error = %v", err)
	}
	if _, err := r2.Read(buf2); err != nil {
		t.Fatalf("Second read error = %v", err)
	}

	if !bytes.Equal(buf1, buf2) {
		t.Error("Same seed produced different random bytes")
	}
}

func TestGetRandReader_WithoutSeed(t *testing.T) {
	t.Parallel()

	r := getRandReader("")

	if r != rand.Reader {
		t.Error("getRandReader(\"\") should return crypto/rand.Reader")
	}
}

func TestGenerateRandomString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		length int
		seed   string
	}{
		{"length 8", 8, "seed1"},
		{"length 16", 16, "seed2"},
		{"length 32", 32, "seed3"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := getRandReader(tc.seed)
			s, err := generateRandomString(tc.length, r)

			if err != nil {
				t.Fatalf("generateRandomString() error = %v", err)
			}
			if len(s) != tc.length {
				t.Errorf("generateRandomString() length = %d, want %d", len(s), tc.length)
			}
		})
	}
}

func TestGenerateRandomString_Deterministic(t *testing.T) {
	t.Parallel()

	seed := "deterministic"
	r1 := getRandReader(seed)
	r2 := getRandReader(seed)

	s1, err1 := generateRandomString(16, r1)
	if err1 != nil {
		t.Fatalf("First generateRandomString() error = %v", err1)
	}

	s2, err2 := generateRandomString(16, r2)
	if err2 != nil {
		t.Fatalf("Second generateRandomString() error = %v", err2)
	}

	if s1 != s2 {
		t.Errorf("Same seed produced different strings: %q vs %q", s1, s2)
	}
}

func TestDRand_Read(t *testing.T) {
	t.Parallel()

	dr := newDRand("test-seed")

	buf := make([]byte, 64)
	n, err := dr.Read(buf)

	if err != nil {
		t.Fatalf("dRand.Read() error = %v", err)
	}
	if n != 64 {
		t.Errorf("dRand.Read() read %d bytes, want 64", n)
	}
}

func TestDRand_Read_SingleByte(t *testing.T) {
	t.Parallel()

	dr := newDRand("test-seed")

	buf := make([]byte, 1)
	_, err := dr.Read(buf)

	// Single byte reads should return an error (workaround for Go's non-determinism check)
	if err == nil {
		t.Error("dRand.Read() with 1 byte should return error")
	}
}

func TestDRand_Deterministic(t *testing.T) {
	t.Parallel()

	seed := "same-seed"

	dr1 := newDRand(seed)
	buf1 := make([]byte, 32)
	dr1.Read(buf1)

	dr2 := newDRand(seed)
	buf2 := make([]byte, 32)
	dr2.Read(buf2)

	if !bytes.Equal(buf1, buf2) {
		t.Error("Same seed produced different deterministic random bytes")
	}
}

func TestDRand_MultipleCycles(t *testing.T) {
	t.Parallel()

	dr := newDRand("test-seed")

	// Read more than one hash cycle worth of data
	buf := make([]byte, 128) // More than sha512.Size/2 (32 bytes)
	n, err := dr.Read(buf)

	if err != nil {
		t.Fatalf("dRand.Read() error = %v", err)
	}
	if n != 128 {
		t.Errorf("dRand.Read() read %d bytes, want 128", n)
	}
}
