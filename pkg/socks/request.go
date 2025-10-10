package socks

import (
	"fmt"
	"io"
	"net"
	"net/netip"
)

// ################### Request ######################### //
//
// https://datatracker.ietf.org/doc/html/rfc1928#section-4
//
//        +----+-----+-------+------+----------+----------+
//        |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
//        +----+-----+-------+------+----------+----------+
//        | 1  |  1  | X'00' |  1   | Variable |    2     |
//        +----+-----+-------+------+----------+----------+

// Request is a SOCKS request in which a client specifies the command as well as destination host and port
type Request struct {
	Ver     byte
	Cmd     Cmd
	DstAddr Addr
	DstPort uint16
}

// ReadRequest reads a complete SOCKS request from r.
func ReadRequest(r io.Reader) (*Request, error) {
	var out Request
	var err error

	b := []byte{0}
	if _, err = r.Read(b); err != nil {
		return nil, fmt.Errorf("parsing version: %s", err)
	}
	out.Ver = b[0]

	if out.Ver != VersionSocks5 {
		return nil, fmt.Errorf("requested version was %d but only SOCKS5 (%d) supported", out.Ver, VersionSocks5)
	}

	if _, err = r.Read(b); err != nil {
		return nil, fmt.Errorf("parsing command: %s", err)
	}

	switch b[0] {
	case byte(CommandConnect):
		out.Cmd = CommandConnect
	case byte(CommandAssociate):
		out.Cmd = CommandAssociate
	default:
		return nil, ErrCommandNotSupported
	}

	if _, err = r.Read(b); err != nil {
		return nil, fmt.Errorf("parsing reserved (RSV): %s", err)
	}
	if b[0] != RSV {
		return nil, fmt.Errorf("parsing reserved (RSV): unexpected value: %x != %x", b, RSV)
	}

	out.DstAddr, out.DstPort, err = parseAddrAndPort(r)
	if err != nil {
		return nil, fmt.Errorf("parsing address and port: %s", err)
	}

	return &out, nil
}

// DstToUDPAddr converts the destination address and port into a net.UDPAddr.
func (r *Request) DstToUDPAddr() (*net.UDPAddr, error) {
	ip := r.DstAddr.ToNetipAddr()
	if (ip == netip.Addr{}) {
		return nil, fmt.Errorf("%s.ToNetipAddr() is nil", r.DstAddr)
	}

	ap := netip.AddrPortFrom(ip, r.DstPort)
	if (ap == netip.AddrPort{}) {
		return nil, fmt.Errorf("netip.AddrPortFrom(%s, %d) is empty", ip, r.DstPort)
	}

	if !ap.IsValid() {
		return nil, fmt.Errorf("%s is invalid", ap)
	}

	return net.UDPAddrFromAddrPort(ap), nil
}

// ######## Response ######## //
//
// https://datatracker.ietf.org/doc/html/rfc1928#section-4
//
//        +----+-----+-------+------+----------+----------+
//        |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
//        +----+-----+-------+------+----------+----------+
//        | 1  |  1  | X'00' |  1   | Variable |    2     |
//        +----+-----+-------+------+----------+----------+

// Reply is the server's response to the Request.
// In Rep, the server indicates if the connection is a success, or what kind of error was encountered.
// It also communicates host and port values, whose meaning depends on the command previously selected by the client.
type Reply struct {
	Ver     byte
	Rep     Rep
	BndAddr Addr
	BndPort int
}

func (r Reply) atyp() Atyp {
	if r.BndAddr != nil {
		return r.BndAddr.Atyp()
	}

	return 0x0 // not a valid atyp, interpret zero as error
}

func (sr Reply) serialize() []byte {
	var out []byte

	out = append(out, VersionSocks5, byte(sr.Rep), RSV, byte(sr.atyp()))
	out = append(out, sr.BndAddr.Bytes()...)
	out = append(out, byte(sr.BndPort>>8), byte(sr.BndPort))

	return out
}

// WriteReplySuccess writes a complete success reply to w.
// Since we only support the command CONNECT for now, this is always a success reply to that command.
// CONNECT replies contain the SOCKS server's IP and port which it has bound for the connection.
// In our case, this is the local address of the slave bound for the connection.
func WriteReplySuccessConnect(w io.Writer, localAddr net.Addr) error {
	var a Addr
	var atyp Atyp
	var ap netip.AddrPort

	if tcpAddr, ok := localAddr.(*net.TCPAddr); ok && tcpAddr != nil {
		ap = tcpAddr.AddrPort()
	} else if udpAddr, ok := localAddr.(*net.UDPAddr); ok && udpAddr != nil {
		ap = udpAddr.AddrPort()
	} else {
		return fmt.Errorf("address has unexpected type, neither TCP nor UDP: %s", localAddr)
	}

	addr := ap.Addr()
	port := ap.Port()

	if addr.Is4() {
		atyp = AddressTypeIPv4
		a = addrIPv4{IP: addr}
	} else if addr.Is6() {
		atyp = AddressTypeIPv6
		a = addrIPv6{IP: addr}
	} else {
		return fmt.Errorf("IP %s was neither IPv4 nor IPv6", addr)
	}

	return writeReply(w, ReplySuccess, atyp, a, int(port))
}

// WriteReplyError writes a complete error reply to w.
// The error code is contained in rep.
func WriteReplyError(w io.Writer, rep Rep) error {
	return writeReply(w, rep, AddressTypeIPv4, addrIPv4{IP: netip.Addr{}}, 0)
}

func writeReply(w io.Writer, rep Rep, atyp Atyp, bndAddr Addr, bndPort int) error {
	resp := Reply{
		Ver:     VersionSocks5,
		Rep:     rep,
		BndAddr: bndAddr,
		BndPort: bndPort,
	}

	_, err := w.Write(resp.serialize())
	if err != nil {
		return fmt.Errorf("writing serialized reply: %s", err)
	}

	return nil
}
