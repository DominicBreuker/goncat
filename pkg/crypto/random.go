package crypto

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"io"
)

// GenerateRandomString ...
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}

func newDRand(seed string) io.Reader {
	return &dRand{next: []byte(seed)}
}

type dRand struct {
	next []byte
}

func (d *dRand) cycle() []byte {
	result := sha512.Sum512(d.next)
	d.next = result[:sha512.Size/2]
	return result[sha512.Size/2:]
}

func (d *dRand) Read(b []byte) (int, error) {
	n := 0
	for n < len(b) {
		out := d.cycle()
		n += copy(b[n:], out)
	}
	return n, nil
}
