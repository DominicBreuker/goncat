package udp

import (
	"net"
	"testing"
)

func TestStreamConn_ImplementsNetConn(t *testing.T) {
	// This test verifies that StreamConn implements net.Conn interface at compile time
	var _ net.Conn = &StreamConn{}
}
