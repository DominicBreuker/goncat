package net

import (
"context"
"errors"
"net"
"sync"
"testing"
"time"

"dominicbreuker/goncat/pkg/config"
"dominicbreuker/goncat/pkg/log"
"dominicbreuker/goncat/pkg/transport"
)

// fakeListener implements transport.Listener for testing.
type fakeListener struct {
serveErr   error
serveCalls int
closed     bool
closeCh    chan struct{}
mu         sync.Mutex
}

func newFakeListener() *fakeListener {
return &fakeListener{
closeCh: make(chan struct{}),
}
}

func (f *fakeListener) Serve(handler transport.Handler) error {
f.mu.Lock()
f.serveCalls++
f.mu.Unlock()

if f.serveErr != nil {
return f.serveErr
}

// Block until closed or error
<-f.closeCh
return net.ErrClosed
}

func (f *fakeListener) Close() error {
f.mu.Lock()
defer f.mu.Unlock()

if !f.closed {
f.closed = true
close(f.closeCh)
}
return nil
}

// Test successful listen for TCP protocol
func TestListenAndServe_TCP_Success(t *testing.T) {
t.Parallel()

ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()

cfg := &config.Shared{
Protocol: config.ProtoTCP,
Host:     "localhost",
Port:     8080,
Logger:   log.NewLogger(false),
}

handler := func(conn net.Conn) error {
return nil
}

// Mock the listener creation
fakeListener := newFakeListener()
deps := &listenDependencies{
createListener: func(ctx context.Context, cfg *config.Shared) (transport.Listener, error) {
return fakeListener, nil
},
wrapWithTLS: func(handler transport.Handler, cfg *config.Shared) (transport.Handler, error) {
return handler, nil
},
}

err := listenAndServe(ctx, cfg, handler, deps)
if err != nil {
t.Fatalf("listenAndServe() error = %v, want nil", err)
}

fakeListener.mu.Lock()
serveCalls := fakeListener.serveCalls
fakeListener.mu.Unlock()

if serveCalls != 1 {
t.Errorf("Serve() called %d times, want 1", serveCalls)
}
}

// Test listener creation failure
func TestListenAndServe_ListenerCreationFails(t *testing.T) {
t.Parallel()

ctx := context.Background()
cfg := &config.Shared{
Protocol: config.ProtoTCP,
Host:     "localhost",
Port:     8080,
Logger:   log.NewLogger(false),
}

handler := func(conn net.Conn) error {
return nil
}

expectedErr := errors.New("listener creation failed")
deps := &listenDependencies{
createListener: func(ctx context.Context, cfg *config.Shared) (transport.Listener, error) {
return nil, expectedErr
},
wrapWithTLS: func(handler transport.Handler, cfg *config.Shared) (transport.Handler, error) {
return handler, nil
},
}

err := listenAndServe(ctx, cfg, handler, deps)
if err == nil {
t.Fatal("listenAndServe() error = nil, want error")
}
}

// Test serve error propagation
func TestListenAndServe_ServeError(t *testing.T) {
t.Parallel()

ctx := context.Background()
cfg := &config.Shared{
Protocol: config.ProtoTCP,
Host:     "localhost",
Port:     8080,
Logger:   log.NewLogger(false),
}

handler := func(conn net.Conn) error {
return nil
}

expectedErr := errors.New("serve failed")
fakeListener := newFakeListener()
fakeListener.serveErr = expectedErr
// Don't close the channel in advance - let Serve() return the error

deps := &listenDependencies{
createListener: func(ctx context.Context, cfg *config.Shared) (transport.Listener, error) {
return fakeListener, nil
},
wrapWithTLS: func(handler transport.Handler, cfg *config.Shared) (transport.Handler, error) {
return handler, nil
},
}

err := listenAndServe(ctx, cfg, handler, deps)
if err == nil {
t.Fatal("listenAndServe() error = nil, want error")
}
}

// Test context cancellation
func TestListenAndServe_ContextCancellation(t *testing.T) {
t.Parallel()

ctx, cancel := context.WithCancel(context.Background())
cfg := &config.Shared{
Protocol: config.ProtoTCP,
Host:     "localhost",
Port:     8080,
Logger:   log.NewLogger(false),
}

handler := func(conn net.Conn) error {
return nil
}

fakeListener := newFakeListener()

deps := &listenDependencies{
createListener: func(ctx context.Context, cfg *config.Shared) (transport.Listener, error) {
return fakeListener, nil
},
wrapWithTLS: func(handler transport.Handler, cfg *config.Shared) (transport.Handler, error) {
return handler, nil
},
}

// Cancel context after short delay
go func() {
time.Sleep(10 * time.Millisecond)
cancel()
}()

err := listenAndServe(ctx, cfg, handler, deps)
// Should return nil on graceful shutdown
if err != nil {
t.Fatalf("listenAndServe() error = %v, want nil", err)
}

// Verify listener was closed
fakeListener.mu.Lock()
closed := fakeListener.closed
fakeListener.mu.Unlock()

if !closed {
t.Error("Listener was not closed on context cancellation")
}
}

// Test TLS wrapping failure
func TestListenAndServe_TLSWrapFails(t *testing.T) {
t.Parallel()

ctx := context.Background()
cfg := &config.Shared{
Protocol: config.ProtoTCP,
Host:     "localhost",
Port:     8080,
SSL:      true,
Logger:   log.NewLogger(false),
}

handler := func(conn net.Conn) error {
return nil
}

fakeListener := newFakeListener()
expectedErr := errors.New("TLS wrap failed")

deps := &listenDependencies{
createListener: func(ctx context.Context, cfg *config.Shared) (transport.Listener, error) {
return fakeListener, nil
},
wrapWithTLS: func(handler transport.Handler, cfg *config.Shared) (transport.Handler, error) {
return nil, expectedErr
},
}

err := listenAndServe(ctx, cfg, handler, deps)
if err == nil {
t.Fatal("listenAndServe() error = nil, want error")
}
}

// Test that benign close errors are recognized
func TestIsServerClosed(t *testing.T) {
t.Parallel()

tests := []struct {
name string
err  error
want bool
}{
{
name: "net.ErrClosed",
err:  net.ErrClosed,
want: true,
},
{
name: "nil error",
err:  nil,
want: false,
},
{
name: "other error",
err:  errors.New("some error"),
want: false,
},
}

for _, tt := range tests {
tt := tt
t.Run(tt.name, func(t *testing.T) {
t.Parallel()
got := isServerClosed(tt.err)
if got != tt.want {
t.Errorf("isServerClosed() = %v, want %v", got, tt.want)
}
})
}
}
