package mux

import (
	"context"
	"dominicbreuker/goncat/pkg/mux/msg"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

// TestOpenSession verifies master session creation.
func TestOpenSession(t *testing.T) {
	t.Parallel()

	// Create connected pipes for client/server
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	// Start server side in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
		if err != nil {
			t.Errorf("AcceptSession() failed: %v", err)
		}
	}()

	// Open master session
	master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("OpenSession() failed: %v", err)
	}
	defer master.Close()

	if master.sess == nil {
		t.Error("master.sess is nil")
	}
	if master.sess.mux == nil {
		t.Error("master.sess.mux is nil")
	}
	if master.enc == nil {
		t.Error("master.enc is nil")
	}
	if master.dec == nil {
		t.Error("master.dec is nil")
	}

	wg.Wait()

}

// TestMasterSession_Close verifies master session close.
func TestMasterSession_Close(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	ready := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
		if err != nil {
			t.Errorf("AcceptSession() failed: %v", err)
			return
		}
		// Wait for master to be ready
		<-ready
		slave.Close()
	}()

	master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("OpenSession() failed: %v", err)
	}

	// Signal slave that master is ready
	close(ready)

	wg.Wait()

	if err := master.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

// TestMasterSession_SendAndReceive verifies message sending and receiving.
func TestMasterSession_SendAndReceive(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	var slaveReceivedMsg msg.Message
	go func() {
		defer wg.Done()
		slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
		if err != nil {
			t.Errorf("AcceptSession() failed: %v", err)
			return
		}
		defer slave.Close()

		// Receive message from master
		slaveReceivedMsg, err = slave.ReceiveContext(context.Background())
		if err != nil {
			t.Errorf("slave.Receive() failed: %v", err)
			return
		}

		// Send response back to master
		if err := slave.SendContext(context.Background(), msg.SocksAssociate{}); err != nil {
			t.Errorf("slave.Send() failed: %v", err)
		}
	}()

	master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("OpenSession() failed: %v", err)
	}
	defer master.Close()

	// Send message to slave
	testMsg := msg.Connect{RemoteHost: "example.com", RemotePort: 80}
	if err := master.SendContext(context.Background(), testMsg); err != nil {
		t.Fatalf("master.Send() failed: %v", err)
	}

	// Receive response from slave
	receivedMsg, err := master.ReceiveContext(context.Background())
	if err != nil {
		t.Fatalf("master.Receive() failed: %v", err)
	}

	if receivedMsg.MsgType() != "SocksAssociate" {
		t.Errorf("master received MsgType = %q; want %q", receivedMsg.MsgType(), "SocksAssociate")
	}

	wg.Wait()

	if slaveReceivedMsg == nil {
		t.Fatal("slave did not receive message")
	}
	if slaveReceivedMsg.MsgType() != "Connect" {
		t.Errorf("slave received MsgType = %q; want %q", slaveReceivedMsg.MsgType(), "Connect")
	}
}

// TestMasterSession_SendAndGetOneChannel verifies sending a message and opening one channel.
func TestMasterSession_SendAndGetOneChannel(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
		if err != nil {
			t.Errorf("AcceptSession() failed: %v", err)
			return
		}
		defer slave.Close()

		// Receive message
		_, err = slave.ReceiveContext(context.Background())
		if err != nil {
			t.Errorf("slave.Receive() failed: %v", err)
			return
		}

		// Accept the channel from master
		conn, err := slave.AcceptNewChannelContext(context.Background())
		if err != nil {
			t.Errorf("slave.AcceptNewChannel() failed: %v", err)
			return
		}
		defer conn.Close()

		// Write some data to verify channel works
		if _, err := conn.Write([]byte("test")); err != nil {
			t.Errorf("conn.Write() failed: %v", err)
		}
	}()

	master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("OpenSession() failed: %v", err)
	}
	defer master.Close()

	testMsg := msg.Connect{RemoteHost: "example.com", RemotePort: 80}
	conn, err := master.SendAndGetOneChannelContext(context.Background(), testMsg)
	if err != nil {
		t.Fatalf("SendAndGetOneChannel() failed: %v", err)
	}
	defer conn.Close()

	// Read data from slave
	buf := make([]byte, 4)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("conn.Read() failed: %v", err)
	}
	if string(buf[:n]) != "test" {
		t.Errorf("conn.Read() = %q; want %q", string(buf[:n]), "test")
	}

	wg.Wait()
}

// TestMasterSession_SendAndGetTwoChannels verifies sending a message and opening two channels.
func TestMasterSession_SendAndGetTwoChannels(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
		if err != nil {
			t.Errorf("AcceptSession() failed: %v", err)
			return
		}
		defer slave.Close()

		// Receive message
		_, err = slave.ReceiveContext(context.Background())
		if err != nil {
			t.Errorf("slave.Receive() failed: %v", err)
			return
		}

		// Accept two channels from master
		conn1, err := slave.AcceptNewChannelContext(context.Background())
		if err != nil {
			t.Errorf("slave.AcceptNewChannel() conn1 failed: %v", err)
			return
		}
		defer conn1.Close()

		conn2, err := slave.AcceptNewChannelContext(context.Background())
		if err != nil {
			t.Errorf("slave.AcceptNewChannel() conn2 failed: %v", err)
			return
		}
		defer conn2.Close()

		// Write to both channels
		if _, err := conn1.Write([]byte("chan1")); err != nil {
			t.Errorf("conn1.Write() failed: %v", err)
		}
		if _, err := conn2.Write([]byte("chan2")); err != nil {
			t.Errorf("conn2.Write() failed: %v", err)
		}
	}()

	master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("OpenSession() failed: %v", err)
	}
	defer master.Close()

	testMsg := msg.Foreground{Exec: "/bin/sh", Pty: true}
	conn1, conn2, err := master.SendAndGetTwoChannelsContext(context.Background(), testMsg)
	if err != nil {
		t.Fatalf("SendAndGetTwoChannels() failed: %v", err)
	}
	defer conn1.Close()
	defer conn2.Close()

	// Read from both channels
	buf1 := make([]byte, 5)
	n1, err := conn1.Read(buf1)
	if err != nil {
		t.Fatalf("conn1.Read() failed: %v", err)
	}
	if string(buf1[:n1]) != "chan1" {
		t.Errorf("conn1.Read() = %q; want %q", string(buf1[:n1]), "chan1")
	}

	buf2 := make([]byte, 5)
	n2, err := conn2.Read(buf2)
	if err != nil {
		t.Fatalf("conn2.Read() failed: %v", err)
	}
	if string(buf2[:n2]) != "chan2" {
		t.Errorf("conn2.Read() = %q; want %q", string(buf2[:n2]), "chan2")
	}

	wg.Wait()
}

// TestMasterSession_GetOneChannel verifies opening a channel without sending a message.
func TestMasterSession_GetOneChannel(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
		if err != nil {
			t.Errorf("AcceptSession() failed: %v", err)
			return
		}
		defer slave.Close()

		// Accept the channel from master
		conn, err := slave.AcceptNewChannelContext(context.Background())
		if err != nil {
			t.Errorf("slave.AcceptNewChannel() failed: %v", err)
			return
		}
		defer conn.Close()

		// Echo data back
		buf := make([]byte, 4)
		n, err := conn.Read(buf)
		if err != nil {
			t.Errorf("conn.Read() failed: %v", err)
			return
		}
		if _, err := conn.Write(buf[:n]); err != nil {
			t.Errorf("conn.Write() failed: %v", err)
		}
	}()

	master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("OpenSession() failed: %v", err)
	}
	defer master.Close()

	conn, err := master.GetOneChannelContext(context.Background())
	if err != nil {
		t.Fatalf("GetOneChannel() failed: %v", err)
	}
	defer conn.Close()

	// Write and read data
	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("conn.Write() failed: %v", err)
	}

	buf := make([]byte, 4)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("conn.Read() failed: %v", err)
	}
	if string(buf[:n]) != "ping" {
		t.Errorf("conn.Read() = %q; want %q", string(buf[:n]), "ping")
	}

	wg.Wait()
}

// TestMasterSession_ConcurrentSends verifies that concurrent sends are properly synchronized.
func TestMasterSession_ConcurrentSends(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	receivedCount := 0
	go func() {
		defer wg.Done()
		slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
		if err != nil {
			t.Errorf("AcceptSession() failed: %v", err)
			return
		}
		defer slave.Close()

		// Receive multiple messages
		for i := 0; i < 10; i++ {
			_, err := slave.ReceiveContext(context.Background())
			if err != nil {
				t.Errorf("slave.Receive() %d failed: %v", i, err)
				return
			}
			receivedCount++
		}
	}()

	master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("OpenSession() failed: %v", err)
	}
	defer master.Close()

	// Send messages concurrently
	var sendWg sync.WaitGroup
	for i := 0; i < 10; i++ {
		sendWg.Add(1)
		go func(n int) {
			defer sendWg.Done()
			testMsg := msg.Connect{RemoteHost: "example.com", RemotePort: 80 + n}
			if err := master.SendContext(context.Background(), testMsg); err != nil {
				t.Errorf("master.Send() %d failed: %v", n, err)
			}
		}(i)
	}

	sendWg.Wait()
	wg.Wait()

	if receivedCount != 10 {
		t.Errorf("received %d messages; want 10", receivedCount)
	}
}

// TestMasterSession_SendAndGetTwoChannels_FirstChannelError verifies cleanup when first channel fails.
// This is a regression test to ensure proper error handling.
func TestMasterSession_SendAndGetTwoChannels_ClosesProperly(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
		if err != nil {
			t.Errorf("AcceptSession() failed: %v", err)
			return
		}
		defer slave.Close()

		_, err = slave.ReceiveContext(context.Background())
		if err != nil {
			t.Errorf("slave.Receive() failed: %v", err)
			return
		}

		// Accept both channels
		conn1, err := slave.AcceptNewChannelContext(context.Background())
		if err != nil {
			t.Errorf("slave.AcceptNewChannel() conn1 failed: %v", err)
			return
		}
		conn1.Close()

		conn2, err := slave.AcceptNewChannelContext(context.Background())
		if err != nil {
			t.Errorf("slave.AcceptNewChannel() conn2 failed: %v", err)
			return
		}
		conn2.Close()
	}()

	master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("OpenSession() failed: %v", err)
	}
	defer master.Close()

	testMsg := msg.Foreground{Exec: "/bin/sh", Pty: true}
	conn1, conn2, err := master.SendAndGetTwoChannelsContext(context.Background(), testMsg)
	if err != nil {
		t.Fatalf("SendAndGetTwoChannels() failed: %v", err)
	}

	// Verify both channels are valid
	if conn1 == nil {
		t.Error("conn1 is nil")
	}
	if conn2 == nil {
		t.Error("conn2 is nil")
	}

	if conn1 != nil {
		conn1.Close()
	}
	if conn2 != nil {
		conn2.Close()
	}

	wg.Wait()
}

// TestOpenSession_ServerClosesEarly verifies error handling when server closes during setup.
func TestOpenSession_ServerClosesEarly(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()

	// Close server immediately
	server.Close()

	// Should fail to open session
	_, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
	if err == nil {
		t.Error("OpenSession() succeeded with closed server; want error")
	}
}

// TestMasterSession_ConcurrentSendAndGetOneChannel verifies that two concurrent
// SendAndGetOneChannel calls from the master each get their own channel and do
// not mix data. This ensures the send+open-channel atomicity is preserved.
func TestMasterSession_ConcurrentSendAndGetOneChannel(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
		if err != nil {
			t.Errorf("AcceptSession() failed: %v", err)
			return
		}
		defer slave.Close()

		// For two expected messages: receive then accept channel and write
		// a distinct payload for each.
		for i := 0; i < 2; i++ {
			_, err := slave.ReceiveContext(context.Background())
			if err != nil {
				t.Errorf("slave.Receive() failed: %v", err)
				return
			}

			conn, err := slave.AcceptNewChannelContext(context.Background())
			if err != nil {
				t.Errorf("slave.AcceptNewChannel() failed: %v", err)
				return
			}

			payload := fmt.Sprintf("resp%d", i)
			if _, err := conn.Write([]byte(payload)); err != nil {
				t.Errorf("conn.Write() failed: %v", err)
			}
			conn.Close()
		}
	}()

	master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("OpenSession() failed: %v", err)
	}
	defer master.Close()

	// Start two concurrent SendAndGetOneChannel calls.
	results := make([]string, 2)
	var sendWg sync.WaitGroup
	sendWg.Add(2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer sendWg.Done()
			testMsg := msg.Connect{RemoteHost: "example.com", RemotePort: 80 + idx}
			conn, err := master.SendAndGetOneChannelContext(context.Background(), testMsg)
			if err != nil {
				t.Errorf("SendAndGetOneChannel() failed: %v", err)
				return
			}
			defer conn.Close()

			buf := make([]byte, 16)
			n, err := conn.Read(buf)
			if err != nil {
				t.Errorf("conn.Read() failed: %v", err)
				return
			}
			results[idx] = string(buf[:n])
		}(i)
	}

	sendWg.Wait()
	wg.Wait()

	// Ensure both expected responses are present.
	found0, found1 := false, false
	for _, r := range results {
		if r == "resp0" {
			found0 = true
		}
		if r == "resp1" {
			found1 = true
		}
	}
	if !found0 || !found1 {
		t.Fatalf("unexpected responses: %v; want resp0 and resp1", results)
	}
}
