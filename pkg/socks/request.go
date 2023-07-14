package socks

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
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
	DstAddr addr
	DstPort int
}

// ReadRequest reads a complete SOCKS request from r
func ReadRequest(r io.Reader) (*Request, error) {
	var out Request

	b := []byte{0}
	if _, err := r.Read(b); err != nil {
		return nil, fmt.Errorf("parsing version: %s", err)
	}
	out.Ver = b[0]

	if out.Ver != VersionSocks5 {
		return nil, fmt.Errorf("requested version was %d but only SOCKS5 (%d) supported", out.Ver, VersionSocks5)
	}

	if _, err := r.Read(b); err != nil {
		return nil, fmt.Errorf("parsing command: %s", err)
	}

	switch b[0] {
	case byte(CommandConnect):
		out.Cmd = CommandConnect
	default:
		return nil, ErrCommandNotSupported
	}

	if _, err := r.Read(b); err != nil {
		return nil, fmt.Errorf("parsing reserved (RSV): %s", err)
	}
	if b[0] != RSV {
		return nil, fmt.Errorf("parsing reserved (RSV): unexpected value: %x != %x", b, RSV)
	}

	if _, err := r.Read(b); err != nil {
		return nil, fmt.Errorf("parsing address type: %s", err)
	}

	switch b[0] {
	case byte(AddressTypeIPv4):
		ip, err := readIPv4(r)
		if err != nil {
			return nil, fmt.Errorf("reading IPv4 address: %s", err)
		}
		out.DstAddr = addrIPv4{IP: ip}
	case byte(AddressTypeFQDN):
		fqdn, err := readFQDN(r)
		if err != nil {
			return nil, fmt.Errorf("reading FQDN address: %s", err)
		}
		out.DstAddr = addrFQDN{FQDN: fqdn}
	case byte(AddressTypeIPv6):
		ip, err := readIPv6(r)
		if err != nil {
			return nil, fmt.Errorf("reading IPv6 address: %s", err)
		}
		out.DstAddr = addrIPv6{IP: ip}
	default:
		return nil, ErrAddressTypeNotSupported
	}

	port := make([]byte, 2)
	if _, err := io.ReadFull(r, port); err != nil {
		return nil, fmt.Errorf("reading port: %s", err)
	}
	out.DstPort = int(binary.BigEndian.Uint16(port))

	return &out, nil
}

func readIPv4(r io.Reader) (net.IP, error) {
	ip := make([]byte, 4) // IPv4
	if _, err := io.ReadFull(r, ip); err != nil {
		return nil, fmt.Errorf("reading ip: %s", err)
	}

	return net.IPv4(ip[0], ip[1], ip[2], ip[3]), nil
}

func readIPv6(r io.Reader) (net.IP, error) {
	ip := make([]byte, 16) // IPv6
	if _, err := io.ReadFull(r, ip); err != nil {
		return nil, fmt.Errorf("reading ip: %s", err)
	}

	return net.IP(ip), nil
}

func readFQDN(r io.Reader) (string, error) {
	size := []byte{0}
	if _, err := r.Read(size); err != nil {
		return "", fmt.Errorf("parsing FQDN size: %s", err)
	}

	fqdn := make([]byte, int(size[0]))
	if _, err := io.ReadFull(r, fqdn); err != nil {
		return "", fmt.Errorf("reading FQDN of size %d: %s", len(fqdn), err)
	}

	return string(fqdn), nil
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
// In Rep, the server indicates if the connection is a success, or what kind of error was encountered.,
// It also communicates host and port values, whose meaning depends on the command previously selected by the client.
type Reply struct {
	Ver     byte
	Rep     Rep
	BndAddr addr
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
func WriteReplySuccess(w io.Writer, localAddr net.Addr) error {
	var ip net.IP
	var a addr
	var port int
	var atyp Atyp

	if tcpAddr, ok := localAddr.(*net.TCPAddr); ok && tcpAddr != nil {
		ip = tcpAddr.IP
		port = tcpAddr.Port
	} else {
		return fmt.Errorf("address has unexpected type, neither TCP nor UDP: %s", localAddr)
	}

	if ip.To4() != nil {
		atyp = AddressTypeIPv4
		a = addrIPv4{IP: ip}
	} else if ip.To16() != nil {
		atyp = AddressTypeIPv6
		a = addrIPv6{IP: ip}
	} else {
		return fmt.Errorf("IP %s was neither IPv4 nor IPv6", ip)
	}

	return writeReply(w, ReplySuccess, atyp, a, port)
}

// WriteReplyError writes a complete error reply to w.
// The error code is contained in rep.
func WriteReplyError(w io.Writer, rep Rep) error {
	return writeReply(w, rep, AddressTypeIPv4, addrIPv4{IP: net.IPv4zero}, 0)
}

func writeReply(w io.Writer, rep Rep, atyp Atyp, bndAddr addr, bndPort int) error {
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
