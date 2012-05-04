package main

import (
	"encoding/binary"
	"errors"
	"net"
)

const strippedPacketOffset = 44

type packet struct {
	Full []byte
	Tcp  []byte
	Dest net.TCPAddr
}

// Dissect the given full IPv6 packet.
func (p *packet) Reset(pkt []byte) error {
	p.Full = pkt

	// Locate the TCP packet
	next := int(p.Full[6])
	nextOff := 40
	for next != 6 {
		if nextOff+1 >= len(p.Full) {
			return errors.New("Overran packet looking for TCP header")
		}
		next = int(p.Full[nextOff])
		nextOff = int(p.Full[nextOff+1])*8 + 8
	}
	p.Tcp = p.Full[nextOff:]
	if len(p.Tcp) < 20 {
		return errors.New("Not enough room for a TCP header")
	}
	p.Dest.IP = p.Full[24:40]
	p.Dest.Port = int(binary.BigEndian.Uint16(p.Tcp[2:4]))

	return nil
}

// Munges the buffer given by Reset() and returns the stripped
// payload.
func (p *packet) Strip() []byte {
	subChecksum(p.Tcp[16:18], oneAdd(p.Full[8:40], p.Tcp[:4]))
	return p.Tcp[4:]
}

// Reassemble the given stripped packet. Expects the stripped packet
// to be written at pkt[strippedPacketOffset:] so that it has room to
// put the rest of the headers back.
func Unstrip(pkt []byte, local, virtual *net.TCPAddr) {
	copy(pkt, []byte{
		0x60, 0, 0, 0, // Version, Traffic Class, Flow Label
		0, 0, // Payload length
		6, // Next protocol
		1, // Hop limit
	})
	binary.BigEndian.PutUint16(pkt[4:6], uint16(len(pkt)-40))
	copy(pkt[8:24], virtual.IP)
	copy(pkt[24:40], local.IP)
	binary.BigEndian.PutUint16(pkt[40:42], uint16(virtual.Port))
	binary.BigEndian.PutUint16(pkt[42:44], uint16(local.Port))
	addChecksum(pkt[56:58], oneAdd(pkt[8:40], pkt[40:44]))
}

var icmp = make([]byte, 1280)
var icmpTypPseudo = []byte{0, 58}

func init() {
	copy(icmp, []byte{
		0x60, 0, 0, 0, // Version, Traffic Class, Flow Label
		0, 0, // Payload Length
		58, // Next Protocol
		1,  // Hop limit
	})
	copy(icmp[40:], []byte{1, 3}) // Address Unreachable
}

// Not thread-safe. Which is fine for us, only the tuntap reader
// thread returns ICMP errors.
func icmpError(src net.IP, pkt []byte) []byte {
	n := copy(icmp[56:], pkt)
	binary.BigEndian.PutUint16(icmp[4:6], uint16(n+16))
	copy(icmp[8:24], src)
	copy(icmp[24:40], src)
	binary.BigEndian.PutUint16(icmp[42:44], ^oneAdd(icmp[4:6], icmp[8:40], icmpTypPseudo, icmp[40:42], icmp[56:]))
	return icmp[:n+56]
}
