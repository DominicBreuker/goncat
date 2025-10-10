package crypto

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
)

// getRandReader returns a deterministic reader if a seed is given, otherwise crypto/rand.Reader.
func getRandReader(seed string) io.Reader {
	if seed != "" {
		return newDRand(seed)
	}

	return rand.Reader
}

// generateRandomString generates a random base64 URL-encoded string of the specified length.
func generateRandomString(length int, r io.Reader) (string, error) {
	bytes := make([]byte, length)
	if _, err := r.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}

// dRand is a deterministic version of rand.Reader
type dRand struct {
	next []byte
}

func newDRand(seed string) io.Reader {
	return &dRand{next: []byte(seed)}
}

func (d *dRand) cycle() []byte {
	result := sha512.Sum512(d.next)
	d.next = result[:sha512.Size/2]
	return result[sha512.Size/2:]
}

func (d *dRand) Read(b []byte) (int, error) {
	// https://github.com/golang/go/issues/58637
	// https://github.com/golang/go/blob/release-branch.go1.20/src/crypto/ecdsa/ecdsa.go#L155
	//
	// Go devs want to make sure that certificate generation is nondeterminstic.
	// To do that, they read a byte off io.Random with 50% probability before generating a certificate.
	// They do not allow to bypass that.
	// This ugly little hack below fixes that problem for now.
	// TODO: revisit this "solution" with every jump to a new Go version since there is no guarantee it keeps working.
	if len(b) == 1 {
		return 0, fmt.Errorf("looks like a randutil.MaybeReadByte call Go devs use to make certificate generation non-determinatic, we don't like that here")
	}

	n := 0
	for n < len(b) {
		out := d.cycle()
		n += copy(b[n:], out)
	}
	return n, nil
}
