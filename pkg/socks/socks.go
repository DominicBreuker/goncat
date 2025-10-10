// Package socks implements SOCKS5 protocol support as defined in RFC 1928.
// It provides functionality for parsing and serializing SOCKS5 messages,
// including method selection, connection requests, and UDP datagrams.
// The implementation supports IPv4, IPv6, and FQDN address types, and
// handles TCP CONNECT and UDP ASSOCIATE commands.
package socks

import (
	"errors"
	"net/netip"
)

// #############################
// ########## Version ##########
// #############################

// VersionSocks5 is the only version we support
const VersionSocks5 = byte(0x05)

// ########## Reserved Fields ##########

// RSV is a null byte to be put into reserved fields in SOCKS messages
const RSV = byte(0x00)

// ############################
// ########## Method ##########
// ############################

// Method is a SOCKS method
type Method byte

// MethodNoAuthenticationRequired is the only method we support.
// MethodNoAcceptableMethods is used in responses to indicate that no method requested is supported.
const (
	MethodNoAuthenticationRequired Method = 0x00
	MethodNoAcceptableMethods      Method = 0xff
)

func isKnownMethod(b byte) bool {
	return b == byte(MethodNoAuthenticationRequired) || b == byte(MethodNoAcceptableMethods)
}

// ##############################
// ########## Commands ##########
// ##############################

// Cmd is a SOCKS command
type Cmd byte

// CommandConnect and ... are the commands we support
const (
	CommandConnect   Cmd = 0x01
	CommandAssociate Cmd = 0x03
)

// ErrCommandNotSupported is an error indicating that the command is not supported
var ErrCommandNotSupported = errors.New("command not supported")

// String returns a human-readable string representation of the command.
func (cmd Cmd) String() string {
	switch cmd {
	case CommandConnect:
		return "CONNECT"
	case CommandAssociate:
		return "UDP ASSOCIATE"
	default:
		return "unexpected"
	}
}

// ###################################
// ########## Address Types ##########
// ###################################

// Atyp is a SOCKS address type
type Atyp byte

// AddressTypeIPv4 and ... are the address types we support
const (
	AddressTypeIPv4 Atyp = 0x01
	AddressTypeFQDN Atyp = 0x03
	AddressTypeIPv6 Atyp = 0x04
)

// ErrAddressTypeNotSupported is an error indicating that the address type is not supported
var ErrAddressTypeNotSupported = errors.New("address type not supported")

// ErrFragmentationNotSupported indicates that a datagram was fragmented, but we don't support that
var ErrFragmentationNotSupported = errors.New("fragmentation not supported")

// Addr represents a SOCKS address, which can be IPv4, IPv6, or FQDN.
// It provides methods to convert between different address representations.
type Addr interface {
	String() string
	Bytes() []byte
	Atyp() Atyp
	ToNetipAddr() netip.Addr
}

type addrIPv4 struct {
	//IP net.IP
	IP netip.Addr
}

func (a addrIPv4) String() string {
	return a.IP.String()
}

func (a addrIPv4) Bytes() []byte {
	return a.IP.AsSlice()
}

func (a addrIPv4) Atyp() Atyp {
	return AddressTypeIPv4
}

func (a addrIPv4) ToNetipAddr() netip.Addr {
	return a.IP
	//ipv4 := ([4]byte)(a.Bytes()) // TODO: could panic, check bounds
	//return netip.AddrFrom4(ipv4)
}

type addrFQDN struct {
	FQDN string
}

func (a addrFQDN) String() string {
	return a.FQDN
}

func (a addrFQDN) Bytes() []byte {
	return []byte(a.FQDN)
}

func (a addrFQDN) Atyp() Atyp {
	return AddressTypeFQDN
}

func (a addrFQDN) ToNetipAddr() netip.Addr {
	return netip.Addr{} // TODO: consider resolving to IP, not sure if required
}

type addrIPv6 struct {
	//IP net.IP
	IP netip.Addr
}

func (a addrIPv6) String() string {
	return a.IP.String()
}

func (a addrIPv6) Bytes() []byte {
	//return a.IP.To16()
	return a.IP.AsSlice()
}

func (a addrIPv6) Atyp() Atyp {
	return AddressTypeIPv6
}

func (a addrIPv6) ToNetipAddr() netip.Addr {
	return a.IP
	//ipv6 := ([16]byte)(a.Bytes()) // TODO: could panic, check bounds
	//return netip.AddrFrom16(ipv6)
}

// #############################
// ########## Replies ##########
// #############################

// Rep is a reply indicating to a SOCKS client if there was an error or not
type Rep byte

// ReplySuccess indicates a success, other values indicate errors
const (
	ReplySuccess                 Rep = 0x00
	ReplyGeneralFailure          Rep = 0x01
	ReplyNetworkUnreachable      Rep = 0x03
	ReplyHostUnreachable         Rep = 0x04
	ReplyConnectionRefused       Rep = 0x06
	ReplyCommandNotSupported     Rep = 0x07
	ReplyAddressTypeNotSupported Rep = 0x08
)
