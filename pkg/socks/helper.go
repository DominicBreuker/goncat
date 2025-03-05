package socks

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
)

func parseAddrAndPort(r io.Reader) (address Addr, port uint16, error error) {
	b := []byte{0}

	if _, err := r.Read(b); err != nil {
		return nil, 0, fmt.Errorf("parsing address type: %s", err)
	}

	switch b[0] {
	case byte(AddressTypeIPv4):
		ip, err := readIPv4(r)
		if err != nil {
			return nil, 0, fmt.Errorf("reading IPv4 address: %s", err)
		}
		address = addrIPv4{IP: ip}
	case byte(AddressTypeFQDN):
		fqdn, err := readFQDN(r)
		if err != nil {
			return nil, 0, fmt.Errorf("reading FQDN address: %s", err)
		}
		address = addrFQDN{FQDN: fqdn}
	case byte(AddressTypeIPv6):
		ip, err := readIPv6(r)
		if err != nil {
			return nil, 0, fmt.Errorf("reading IPv6 address: %s", err)
		}
		address = addrIPv6{IP: ip}
	default:
		return nil, 0, ErrAddressTypeNotSupported
	}

	p := make([]byte, 2)
	if _, err := io.ReadFull(r, p); err != nil {
		return nil, 0, fmt.Errorf("reading port: %s", err)
	}
	port = binary.BigEndian.Uint16(p)

	return address, port, nil
}

// func readIPv4(r io.Reader) (net.IP, error) {
func readIPv4(r io.Reader) (netip.Addr, error) {
	ip := make([]byte, 4) // IPv4
	if _, err := io.ReadFull(r, ip); err != nil {
		return netip.Addr{}, fmt.Errorf("reading ip: %s", err)
	}

	return netip.AddrFrom4(([4]byte)(ip)), nil
}

func readIPv6(r io.Reader) (netip.Addr, error) {
	ip := make([]byte, 16) // IPv6
	if _, err := io.ReadFull(r, ip); err != nil {
		return netip.Addr{}, fmt.Errorf("reading ip: %s", err)
	}

	//return net.IP(ip), nil
	return netip.AddrFrom16(([16]byte)(ip)), nil
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
