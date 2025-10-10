package socks

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
)

// ################### Request ######################### //
//
// https://datatracker.ietf.org/doc/html/rfc1928#section-7
//
//      +----+------+------+----------+----------+----------+
//      |RSV | FRAG | ATYP | DST.ADDR | DST.PORT |   DATA   |
//      +----+------+------+----------+----------+----------+
//      | 2  |  1   |  1   | Variable |    2     | Variable |
//      +----+------+------+----------+----------+----------+

// UDPRequest represents a SOCKS5 UDP datagram as defined in RFC 1928 section 7.
type UDPRequest struct {
	Frag    byte
	DstAddr Addr
	DstPort uint16
	Data    []byte
}

func (r *UDPRequest) String() string {
	return fmt.Sprintf("Datagram[%d|%s:%d|%s]", r.Frag, r.DstAddr, r.DstPort, r.Data) // just for debugging
}

// ReadUDPDatagram parses a SOCKS5 UDP datagram from the provided byte slice.
func ReadUDPDatagram(data []byte) (*UDPRequest, error) {
	var out UDPRequest

	if data[0] != 0 || data[1] != 0 {
		return nil, fmt.Errorf("RSV must be zero but was %x", data[:2])
	}

	out.Frag = data[2]
	if out.Frag > 127 {
		return nil, fmt.Errorf("FRAG must be <= 127 but was %x", out.Frag)
	}

	if out.Frag != 0 {
		return nil, ErrFragmentationNotSupported
	}

	atyp := data[3]

	var addrLen int
	switch atyp {
	case byte(AddressTypeIPv4):
		addrLen = 4
		out.DstAddr = addrIPv4{IP: netip.AddrFrom4(([4]byte)(data[4:8]))}
		out.DstPort = binary.BigEndian.Uint16(data[8:10])
	case byte(AddressTypeFQDN):
		addrLen = int(data[4])
		out.DstAddr = addrFQDN{FQDN: string(data[5 : 5+addrLen])}
		out.DstPort = binary.BigEndian.Uint16(data[5+addrLen : 7+addrLen])

		addrLen += 1 // first octed was length of domain name, account for that when getting offset to data
	case byte(AddressTypeIPv6):
		addrLen = 16
		out.DstAddr = addrIPv6{IP: netip.AddrFrom16(([16]byte)(data[4:20]))}
		out.DstPort = binary.BigEndian.Uint16(data[20:22])
	default:
		return nil, fmt.Errorf("unexpected ATYP %x", atyp)
	}

	out.Data = data[4+addrLen+2:]

	return &out, nil
}

// WriteUDPRequestAddrPort writes a SOCKS5 UDP datagram to the specified address and port.
func WriteUDPRequestAddrPort(conn *net.UDPConn, ip netip.Addr, port uint16, data []byte) error {
	var a Addr

	if ip.Is4() {
		a = addrIPv4{IP: ip}
	} else if ip.Is6() {
		a = addrIPv6{IP: ip}
	} else {
		return fmt.Errorf("IP %s was neither IPv4 nor IPv6", ip)
	}

	req := UDPRequest{
		Frag:    byte(0),
		DstAddr: a,
		DstPort: uint16(port),
		Data:    data,
	}

	_, err := conn.WriteToUDPAddrPort(req.serialize(), netip.AddrPortFrom(ip, port))
	if err != nil {
		return fmt.Errorf("writing serialized reply: %s", err)
	}

	return nil
}

// FRAG is the fragment field value for non-fragmented datagrams.
const FRAG = byte(0x0)

func (r UDPRequest) serialize() []byte {
	var out []byte

	out = append(out, RSV, RSV, FRAG, byte(r.DstAddr.Atyp()))
	out = append(out, r.DstAddr.Bytes()...)
	out = append(out, byte(r.DstPort>>8), byte(r.DstPort))
	out = append(out, r.Data...)

	return out
}
