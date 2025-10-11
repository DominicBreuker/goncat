package mocks

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestMockTCPNetwork_ListenAndDial(t *testing.T) {
	mockNet := NewMockTCPNetwork()

	// Test listener creation
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9001")
	listener, err := mockNet.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("ListenTCP failed: %v", err)
	}
	defer listener.Close()

	// Test duplicate listener
	_, err = mockNet.ListenTCP("tcp", addr)
	if err == nil {
		t.Error("Expected error for duplicate listener address")
	}

	// Test connection establishment
	done := make(chan bool)
	go func() {
		serverConn, err := listener.Accept()
		if err != nil {
			t.Errorf("Accept failed: %v", err)
			return
		}
		defer serverConn.Close()

		buf := make([]byte, 1024)
		n, err := serverConn.Read(buf)
		if err != nil && err != io.EOF {
			t.Errorf("Read failed: %v", err)
			return
		}

		if string(buf[:n]) != "test message" {
			t.Errorf("Expected 'test message', got %q", buf[:n])
		}
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)

	clientAddr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9002")
	clientConn, err := mockNet.DialTCP("tcp", clientAddr, addr)
	if err != nil {
		t.Fatalf("DialTCP failed: %v", err)
	}
	defer clientConn.Close()

	_, err = clientConn.Write([]byte("test message"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Test timeout")
	}
}

func TestMockTCPNetwork_ConnectionRefused(t *testing.T) {
	mockNet := NewMockTCPNetwork()

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9999")
	_, err := mockNet.DialTCP("tcp", nil, addr)
	if err == nil {
		t.Error("Expected connection refused error")
	}
}
