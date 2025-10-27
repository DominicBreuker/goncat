// Package config defines configuration structures and validation logic
// for the goncat application, including protocol definitions, shared settings,
// and master/slave specific configurations.
package config

import (
	"dominicbreuker/goncat/pkg/log"
	"fmt"
	mrand "math/rand"
	"time"
)

// Shared contains configuration settings common to both master and slave modes,
// including network protocol, connection details, and security settings.
type Shared struct {
	ID       string
	Protocol Protocol
	Host     string
	Port     int
	SSL      bool
	Key      string
	Verbose  bool
	Timeout  time.Duration
	Deps     *Dependencies
	Logger   *log.Logger
}

// Protocol represents the network protocol type used for communication.
type Protocol int

// Protocol type constants.
const (
	ProtoTCP = 1 // ProtoTCP represents plain TCP protocol
	ProtoWS  = 2 // ProtoWS represents WebSocket protocol without TLS
	ProtoWSS = 3 // ProtoWSS represents WebSocket protocol with TLS
	ProtoUDP = 4 // ProtoUDP represents UDP protocol with QUIC
)

// String returns the string representation of the Protocol.
func (p Protocol) String() string {
	switch p {
	case ProtoTCP:
		return "tcp"
	case ProtoWS:
		return "ws"
	case ProtoWSS:
		return "wss"
	case ProtoUDP:
		return "udp"
	default:
		return ""
	}
}

// KeySalt is a random salt value used in key derivation.
// This value is overwritten with a random value during release builds via ldflags.
var KeySalt = "98263df478dbb76e25eed7e71750e59dbffcb1f401413472f9b128f10bb3cc01af3942a17980a24cd1a26bd3ab87a0fec835faf59aa4f1a1dc7f2416c5765e9e"

// Validate checks the Shared configuration for errors.
// It returns a slice of validation errors, or an empty slice if valid.
func (c *Shared) Validate() []error {
	var errors []error

	if !c.SSL && c.Key != "" {
		errors = append(errors, fmt.Errorf("you must use '--ssl' to use '--key'"))
	}

	if err := validatePort(c.Port); err != nil {
		errors = append(errors, fmt.Errorf("'--port': %s", err))
	}

	return errors
}

// GetKey returns the salted key for authentication.
// If no key is configured, it returns an empty string.
// Otherwise, it returns the KeySalt concatenated with the configured key.
func (c *Shared) GetKey() string {
	if c.Key == "" {
		return ""
	}

	return KeySalt + c.Key
}

// GenerateId returns a pseudo-random 12-character alphanumeric string for
// non-security uses (e.g., logging). It never returns an error.
func GenerateId() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 12

	src := mrand.NewSource(time.Now().UnixNano())
	r := mrand.New(src)

	buf := make([]byte, length)
	for i := 0; i < length; i++ {
		buf[i] = letters[r.Intn(len(letters))]
	}

	return string(buf)
}
