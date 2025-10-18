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

// TestAcceptSession verifies slave session creation.
func TestAcceptSession(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	// Start client side in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
		if err != nil {
			t.Errorf("OpenSession() failed: %v", err)
		}
	}()

	// Accept slave session
	slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("AcceptSession() failed: %v", err)
	}
	defer slave.Close()

	if slave.sess == nil {
		t.Error("slave.sess is nil")
	}
	if slave.sess.mux == nil {
		t.Error("slave.sess.mux is nil")
	}
	if slave.enc == nil {
		t.Error("slave.enc is nil")
	}
	if slave.dec == nil {
		t.Error("slave.dec is nil")
	}

	wg.Wait()
}

// TestSlaveSession_Close verifies slave session close.
func TestSlaveSession_Close(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	ready := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
		if err != nil {
			t.Errorf("OpenSession() failed: %v", err)
			return
		}
		// Wait for slave to be ready
		<-ready
		master.Close()
	}()

	slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("AcceptSession() failed: %v", err)
	}

	// Signal master that slave is ready
	close(ready)

	wg.Wait()

	if err := slave.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

// TestSlaveSession_SendAndReceive verifies message sending and receiving from slave side.
func TestSlaveSession_SendAndReceive(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	var masterReceivedMsg msg.Message
	go func() {
		defer wg.Done()
		master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
		if err != nil {
			t.Errorf("OpenSession() failed: %v", err)
			return
		}
		defer master.Close()

		// Send message to slave
		if err := master.SendContext(context.Background(), msg.Connect{RemoteHost: "test.com", RemotePort: 443}); err != nil {
			t.Errorf("master.Send() failed: %v", err)
			return
		}

		// Receive response from slave
		masterReceivedMsg, err = master.ReceiveContext(context.Background())
		if err != nil {
			t.Errorf("master.Receive() failed: %v", err)
		}
	}()

	slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("AcceptSession() failed: %v", err)
	}
	defer slave.Close()

	// Receive message from master
	receivedMsg, err := slave.ReceiveContext(context.Background())
	if err != nil {
		t.Fatalf("slave.Receive() failed: %v", err)
	}

	if receivedMsg.MsgType() != "Connect" {
		t.Errorf("slave received MsgType = %q; want %q", receivedMsg.MsgType(), "Connect")
	}

	// Send response to master
	testMsg := msg.SocksConnect{RemoteHost: "response.com", RemotePort: 8080}
	if err := slave.SendContext(context.Background(), testMsg); err != nil {
		t.Fatalf("slave.Send() failed: %v", err)
	}

	wg.Wait()

	if masterReceivedMsg == nil {
		t.Fatal("master did not receive message")
	}
	if masterReceivedMsg.MsgType() != "SocksConnect" {
		t.Errorf("master received MsgType = %q; want %q", masterReceivedMsg.MsgType(), "SocksConnect")
	}
}

// TestSlaveSession_SendAndGetOneChannel verifies sending a message and accepting one channel.
func TestSlaveSession_SendAndGetOneChannel(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	// ready is signaled by master after it has opened the data stream and is
	// prepared to read. This avoids a timing race where the slave writes while
	// the session is being shut down or not yet fully ready.
	ready := make(chan struct{})

	go func() {
		defer wg.Done()
		master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
		if err != nil {
			t.Errorf("OpenSession() failed: %v", err)
			return
		}
		defer master.Close()

		// Receive message from slave
		_, err = master.ReceiveContext(context.Background())
		if err != nil {
			t.Errorf("master.Receive() failed: %v", err)
			return
		}

		// Open a channel for slave to accept
		conn, err := master.GetOneChannelContext(context.Background())
		if err != nil {
			t.Errorf("master.openNewChannel() failed: %v", err)
			return
		}
		defer conn.Close()

		// Signal the slave goroutine that master is ready to read.
		close(ready)

		// Read data from slave
		buf := make([]byte, 4)
		n, err := conn.Read(buf)
		if err != nil {
			t.Errorf("conn.Read() failed: %v", err)
			return
		}
		if string(buf[:n]) != "data" {
			t.Errorf("conn.Read() = %q; want %q", string(buf[:n]), "data")
		}
	}()

	slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("AcceptSession() failed: %v", err)
	}
	defer slave.Close()

	testMsg := msg.SocksConnect{RemoteHost: "example.com", RemotePort: 1080}
	conn, err := slave.SendAndGetOneChannelContext(context.Background(), testMsg)
	if err != nil {
		t.Fatalf("SendAndGetOneChannel() failed: %v", err)
	}
	defer conn.Close()

	// Wait for master to be ready, then write data to master
	<-ready
	if _, err := conn.Write([]byte("data")); err != nil {
		t.Fatalf("conn.Write() failed: %v", err)
	}

	wg.Wait()
}

// TestSlaveSession_GetOneChannel verifies accepting a channel without sending a message.
func TestSlaveSession_GetOneChannel(t *testing.T) {
	// Skip this flaky test in short mode.
	if testing.Short() {
		t.Skip("skipping flaky test in short mode")
	}

	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	// Use a channel to synchronize
	ready := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
		if err != nil {
			t.Errorf("OpenSession() failed: %v", err)
			return
		}
		defer master.Close()

		// Signal slave is ready
		<-ready

		// Open a channel for slave
		conn, err := master.GetOneChannelContext(context.Background())
		if err != nil {
			t.Errorf("master.openNewChannel() failed: %v", err)
			return
		}
		defer conn.Close()

		// Write data
		if _, err := conn.Write([]byte("test")); err != nil {
			t.Errorf("conn.Write() failed: %v", err)
		}
	}()

	slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("AcceptSession() failed: %v", err)
	}
	defer slave.Close()

	// Signal master to proceed
	close(ready)

	conn, err := slave.GetOneChannelContext(context.Background())
	if err != nil {
		t.Fatalf("GetOneChannel() failed: %v", err)
	}
	defer conn.Close()

	// Read data from master
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

// TestSlaveSession_AcceptNewChannel verifies accepting a new channel.
func TestSlaveSession_AcceptNewChannel(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
		if err != nil {
			t.Errorf("OpenSession() failed: %v", err)
			return
		}
		defer master.Close()

		// Open multiple channels and keep them open until slave accepts them
		var conns []net.Conn
		for i := 0; i < 3; i++ {
			conn, err := master.GetOneChannelContext(context.Background())
			if err != nil {
				t.Errorf("master.openNewChannel() %d failed: %v", i, err)
				return
			}
			conns = append(conns, conn)
		}

		// Wait for slave to signal it's done accepting channels
		<-done

		// Now close all channels
		for _, conn := range conns {
			conn.Close()
		}
	}()

	slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("AcceptSession() failed: %v", err)
	}
	defer slave.Close()

	// Accept multiple channels
	for i := 0; i < 3; i++ {
		conn, err := slave.AcceptNewChannelContext(context.Background())
		if err != nil {
			t.Fatalf("AcceptNewChannel() %d failed: %v", i, err)
		}
		conn.Close()
	}

	// Signal that we're done accepting channels
	close(done)

	wg.Wait()
}

// TestSlaveSession_ConcurrentReceives verifies that concurrent receives work correctly.
func TestSlaveSession_ConcurrentReceives(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	messageCount := 10
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
		if err != nil {
			t.Errorf("OpenSession() failed: %v", err)
			return
		}
		defer master.Close()

		// Send multiple messages
		for i := 0; i < messageCount; i++ {
			testMsg := msg.Connect{RemoteHost: "example.com", RemotePort: 80 + i}
			if err := master.SendContext(context.Background(), testMsg); err != nil {
				t.Errorf("master.Send() %d failed: %v", i, err)
				return
			}
		}

		// Wait for slave to finish receiving messages before closing
		<-done
	}()

	slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("AcceptSession() failed: %v", err)
	}
	defer slave.Close()

	// Receive messages
	receivedCount := 0
	for i := 0; i < messageCount; i++ {
		_, err = slave.ReceiveContext(context.Background())
		if err != nil {
			t.Fatalf("slave.Receive() %d failed: %v", i, err)
		}
		receivedCount++
	}

	if receivedCount != messageCount {
		t.Errorf("received %d messages; want %d", receivedCount, messageCount)
	}

	// Signal we're done before waiting
	close(done)

	wg.Wait()
}

// TestAcceptSession_ClientClosesEarly verifies error handling when client closes during setup.
func TestAcceptSession_ClientClosesEarly(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer server.Close()

	// Close client immediately
	client.Close()

	// Should fail to accept session
	_, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
	if err == nil {
		t.Error("AcceptSession() succeeded with closed client; want error")
	}
}

// TestSlaveSession_MultipleChannelOperations verifies multiple channel operations work together.
func TestSlaveSession_MultipleChannelOperations(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
		if err != nil {
			t.Errorf("OpenSession() failed: %v", err)
			return
		}
		defer master.Close()

		// Receive message and open channels
		_, err = master.ReceiveContext(context.Background())
		if err != nil {
			t.Errorf("master.Receive() failed: %v", err)
			return
		}

		conn, err := master.GetOneChannelContext(context.Background())
		if err != nil {
			t.Errorf("master.openNewChannel() failed: %v", err)
			return
		}
		defer conn.Close()

		// Exchange data
		if _, err := conn.Write([]byte("hello")); err != nil {
			t.Errorf("conn.Write() failed: %v", err)
			return
		}

		buf := make([]byte, 5)
		n, err := conn.Read(buf)
		if err != nil {
			t.Errorf("conn.Read() failed: %v", err)
			return
		}
		if string(buf[:n]) != "world" {
			t.Errorf("conn.Read() = %q; want %q", string(buf[:n]), "world")
		}
	}()

	slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("AcceptSession() failed: %v", err)
	}
	defer slave.Close()

	testMsg := msg.PortFwd{LocalHost: "localhost", LocalPort: 8080, RemoteHost: "remote", RemotePort: 9090}
	conn, err := slave.SendAndGetOneChannelContext(context.Background(), testMsg)
	if err != nil {
		t.Fatalf("SendAndGetOneChannel() failed: %v", err)
	}
	defer conn.Close()

	// Exchange data
	buf := make([]byte, 5)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("conn.Read() failed: %v", err)
	}
	if string(buf[:n]) != "hello" {
		t.Errorf("conn.Read() = %q; want %q", string(buf[:n]), "hello")
	}

	if _, err := conn.Write([]byte("world")); err != nil {
		t.Fatalf("conn.Write() failed: %v", err)
	}

	wg.Wait()
}

// TestSlaveSession_ConcurrentSendAndGetOneChannel mirrors the master-side test
// to ensure that the slave's SendAndGetOneChannel operations do not mix
// channels when running concurrently.
func TestSlaveSession_ConcurrentSendAndGetOneChannel(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		master, err := OpenSessionContext(context.Background(), client, 50*time.Millisecond)
		if err != nil {
			t.Errorf("OpenSession() failed: %v", err)
			return
		}
		defer master.Close()

		// For two expected messages: receive then open channel and write
		// a distinct payload for each.
		for i := 0; i < 2; i++ {
			_, err := master.ReceiveContext(context.Background())
			if err != nil {
				t.Errorf("master.Receive() failed: %v", err)
				return
			}

			conn, err := master.GetOneChannelContext(context.Background())
			if err != nil {
				t.Errorf("master.openNewChannel() failed: %v", err)
				return
			}

			payload := fmt.Sprintf("mresp%d", i)
			if _, err := conn.Write([]byte(payload)); err != nil {
				t.Errorf("conn.Write() failed: %v", err)
			}
			conn.Close()
		}
	}()

	slave, err := AcceptSessionContext(context.Background(), server, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("AcceptSession() failed: %v", err)
	}
	defer slave.Close()

	// Start two concurrent slave SendAndGetOneChannel calls.
	results := make([]string, 2)
	var sendWg sync.WaitGroup
	sendWg.Add(2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			defer sendWg.Done()
			testMsg := msg.Connect{RemoteHost: "example.com", RemotePort: 80 + idx}
			conn, err := slave.SendAndGetOneChannelContext(context.Background(), testMsg)
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
		if r == "mresp0" {
			found0 = true
		}
		if r == "mresp1" {
			found1 = true
		}
	}
	if !found0 || !found1 {
		t.Fatalf("unexpected responses: %v; want mresp0 and mresp1", results)
	}
}
