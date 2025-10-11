package config

import (
	"io"
	"net"
	"os"
	"os/exec"
)

// Dependencies contains injectable dependencies for testing and customization.
// All fields are optional and will use default implementations if nil.
type Dependencies struct {
	TCPDialer        TCPDialerFunc
	TCPListener      TCPListenerFunc
	PortFwdListener  PortFwdListenerFunc
	PortFwdDialer    PortFwdDialerFunc
	Stdin            StdinFunc
	Stdout           StdoutFunc
	ExecCommand      ExecCommandFunc
}

// TCPDialerFunc is a function that dials a TCP connection.
// It returns a net.Conn to allow for mock implementations.
type TCPDialerFunc func(network string, laddr, raddr *net.TCPAddr) (net.Conn, error)

// TCPListenerFunc is a function that creates a TCP listener.
// It returns a net.Listener to allow for mock implementations.
type TCPListenerFunc func(network string, laddr *net.TCPAddr) (net.Listener, error)

// PortFwdListenerFunc is a function that creates a listener for port forwarding.
// It returns a net.Listener to allow for mock implementations in port forwarding scenarios.
type PortFwdListenerFunc func(network string, laddr *net.TCPAddr) (net.Listener, error)

// PortFwdDialerFunc is a function that dials a connection for port forwarding.
// It returns a net.Conn to allow for mock implementations in port forwarding scenarios.
type PortFwdDialerFunc func(network string, laddr, raddr *net.TCPAddr) (net.Conn, error)

// StdinFunc is a function that returns a reader for stdin.
// It returns an io.Reader to allow for mock implementations.
type StdinFunc func() io.Reader

// StdoutFunc is a function that returns a writer for stdout.
// It returns an io.Writer to allow for mock implementations.
type StdoutFunc func() io.Writer

// ExecCommandFunc is a function that creates a command executor.
// It returns a Cmd interface to allow for mock implementations.
type ExecCommandFunc func(program string) Cmd

// Cmd is an interface that represents a command to be executed.
// It wraps the functionality needed from *exec.Cmd for testing.
type Cmd interface {
	StdoutPipe() (io.ReadCloser, error)
	StdinPipe() (io.WriteCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
	Process() Process
}

// Process is an interface that represents an OS process.
type Process interface {
	Kill() error
}

// GetTCPDialerFunc returns the TCP dialer function from dependencies, or a default implementation.
// If deps is nil or deps.TCPDialer is nil, returns a function that uses net.DialTCP.
func GetTCPDialerFunc(deps *Dependencies) TCPDialerFunc {
	if deps != nil && deps.TCPDialer != nil {
		return deps.TCPDialer
	}
	return func(network string, laddr, raddr *net.TCPAddr) (net.Conn, error) {
		return net.DialTCP(network, laddr, raddr)
	}
}

// GetTCPListenerFunc returns the TCP listener function from dependencies, or a default implementation.
// If deps is nil or deps.TCPListener is nil, returns a function that uses net.ListenTCP.
func GetTCPListenerFunc(deps *Dependencies) TCPListenerFunc {
	if deps != nil && deps.TCPListener != nil {
		return deps.TCPListener
	}
	return func(network string, laddr *net.TCPAddr) (net.Listener, error) {
		return net.ListenTCP(network, laddr)
	}
}

// GetStdinFunc returns the stdin function from dependencies, or a default implementation.
// If deps is nil or deps.Stdin is nil, returns a function that uses os.Stdin.
func GetStdinFunc(deps *Dependencies) StdinFunc {
	if deps != nil && deps.Stdin != nil {
		return deps.Stdin
	}
	return func() io.Reader {
		return os.Stdin
	}
}

// GetStdoutFunc returns the stdout function from dependencies, or a default implementation.
// If deps is nil or deps.Stdout is nil, returns a function that uses os.Stdout.
func GetStdoutFunc(deps *Dependencies) StdoutFunc {
	if deps != nil && deps.Stdout != nil {
		return deps.Stdout
	}
	return func() io.Writer {
		return os.Stdout
	}
}

// GetExecCommandFunc returns the exec command function from dependencies, or a default implementation.
// If deps is nil or deps.ExecCommand is nil, returns a function that uses exec.Command.
func GetExecCommandFunc(deps *Dependencies) ExecCommandFunc {
	if deps != nil && deps.ExecCommand != nil {
		return deps.ExecCommand
	}
	return func(program string) Cmd {
		return &realCmd{cmd: exec.Command(program)}
	}
}

// GetPortFwdListenerFunc returns the port forwarding listener function from dependencies, or a default implementation.
// If deps is nil or deps.PortFwdListener is nil, returns a function that uses net.ListenTCP.
func GetPortFwdListenerFunc(deps *Dependencies) PortFwdListenerFunc {
	if deps != nil && deps.PortFwdListener != nil {
		return deps.PortFwdListener
	}
	return func(network string, laddr *net.TCPAddr) (net.Listener, error) {
		return net.ListenTCP(network, laddr)
	}
}

// GetPortFwdDialerFunc returns the port forwarding dialer function from dependencies, or a default implementation.
// If deps is nil or deps.PortFwdDialer is nil, returns a function that uses net.DialTCP.
func GetPortFwdDialerFunc(deps *Dependencies) PortFwdDialerFunc {
	if deps != nil && deps.PortFwdDialer != nil {
		return deps.PortFwdDialer
	}
	return func(network string, laddr, raddr *net.TCPAddr) (net.Conn, error) {
		return net.DialTCP(network, laddr, raddr)
	}
}

// realCmd wraps *exec.Cmd to implement the Cmd interface.
type realCmd struct {
	cmd *exec.Cmd
}

func (r *realCmd) StdoutPipe() (io.ReadCloser, error) {
	return r.cmd.StdoutPipe()
}

func (r *realCmd) StdinPipe() (io.WriteCloser, error) {
	return r.cmd.StdinPipe()
}

func (r *realCmd) StderrPipe() (io.ReadCloser, error) {
	return r.cmd.StderrPipe()
}

func (r *realCmd) Start() error {
	return r.cmd.Start()
}

func (r *realCmd) Wait() error {
	return r.cmd.Wait()
}

func (r *realCmd) Process() Process {
	return &realProcess{process: r.cmd.Process}
}

// realProcess wraps *os.Process to implement the Process interface.
type realProcess struct {
	process *os.Process
}

func (r *realProcess) Kill() error {
	return r.process.Kill()
}
