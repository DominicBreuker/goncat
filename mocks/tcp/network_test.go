package tcp

import (
	"testing"
)

func TestMockNetworkEcho(t *testing.T) {
	mockNet := NewMockTCPNetwork()

	// Start server on a fixed test port
	srv, err := NewServer(mockNet.ListenTCP, "tcp", "127.0.0.1:9001", "ECHO: ")
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Close()

	var l *MockTCPListener
	if l, err = mockNet.WaitForListener("127.0.0.1:9001", 500); err != nil {
		t.Fatalf("Server failed to start listening: %v", err)
	}

	client, err := NewClient(mockNet.DialTCP, "tcp", "127.0.0.1:9001")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if _, err := l.WaitForNewConnection(500); err != nil {
		t.Fatalf("Client failed to connect: %v", err)
	}

	msgs := []string{"hello", "world", "third"}
	for _, m := range msgs {
		if err := client.WriteLine(m); err != nil {
			t.Fatalf("WriteLine failed: %v", err)
		}
		got, err := client.ReadLine()
		if err != nil {
			t.Fatalf("ReadLine failed: %v", err)
		}
		want := "ECHO: " + m
		if got != want {
			t.Fatalf("unexpected response: got=%q want=%q", got, want)
		}
	}
}
