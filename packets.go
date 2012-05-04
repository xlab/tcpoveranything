package main

import (
	"encoding/binary"
	"net"

	"code.google.com/p/tuntap"
)

// Extract and return the dst TCPAddr from the packet, as well as the
// slimmed down TCP payload to forward and the adjusted checksum
// (which you need to add back at pkt[12:14]. Returns nils if decoding
// screws up.
func mungePacket(p *tuntap.Packet) (*net.TCPAddr, []byte, uint16) {
	if p.Protocol != 0x86dd {
		return nil, nil, 0
	}

	destIp := net.IP(p.Packet[24:40])

	next := int(p.Packet[6])
	nextOff := 40
	for next != 6 {
		if nextOff+1 >= len(p.Packet) {
			return nil, nil, 0
		}
		next = int(p.Packet[nextOff])
		nextOff = int(p.Packet[nextOff+1])*8 + 8
	}
	// nextOff points to the start of the TCP header
	if len(p.Packet)-nextOff < 20 {
		// Not enough room for a TCP header
		return nil, nil, 0
	}

	destPort := int(binary.BigEndian.Uint16(p.Packet[nextOff+2 : nextOff+4]))

	// Compute the adjusted checksum (with the stuff we'll munge
	// substracted)
	sum := &onesComplement{uint64(^binary.BigEndian.Uint16(p.Packet[nextOff+16 : nextOff+18]))}
	sum.Sub(p.Packet[8:40])
	sum.Sub(p.Packet[nextOff : nextOff+4])

	return &net.TCPAddr{destIp, destPort}, p.Packet[nextOff+4:], sum.Sum()
}

var ipv6Header = []byte{
	0x60, 0, 0, 0, // Version, Traffic Class, Flow Label
	0, 0, // Payload length
	0,                                              // Next protocol
	1,                                              // Hop limit (makes sure we stay on localhost)
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // src
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // dst
}

const unMungeStart = 44

// Assumes that the packet is at b[unMungeStart:]
func unMungePacket(b []byte, bind *binding) *tuntap.Packet {
	copy(b, ipv6Header)
	// Payload length
	binary.BigEndian.PutUint16(b[4:6], uint16(len(b)-len(ipv6Header)))
	// Next protocol
	b[6] = 6
	// Source IP
	copy(b[8:24], bind.virtual.IP)
	// Destination IP
	copy(b[24:40], bind.local.IP)
	// Source TCP port
	binary.BigEndian.PutUint16(b[40:42], uint16(bind.virtual.Port))
	// Destination TCP port
	binary.BigEndian.PutUint16(b[42:44], uint16(bind.local.Port))

	// Readjust the TCP checksum for the new data
	sum := &onesComplement{uint64(^binary.BigEndian.Uint16(b[56:58]))}
	sum.Add(b[8:44])
	binary.BigEndian.PutUint16(b[56:58], sum.Sum())

	return &tuntap.Packet{Protocol: 0x86dd, Packet: b}
}

var icmpGoAway = []byte{
	1, 3, // Address unreachable
	0, 0, // Checksum
	0, 0, 0, 0, // Reserved
}

func icmpError(src net.IP, orig []byte) *tuntap.Packet {
	b := make([]byte, 1280)
	copy(b, ipv6Header)
	copy(b[len(ipv6Header):], icmpGoAway)
	// Original payload
	n := copy(b[len(ipv6Header)+len(icmpGoAway):], orig)
	b = b[:len(ipv6Header)+len(icmpGoAway)+n]
	// Payload length
	binary.BigEndian.PutUint16(b[4:6], uint16(len(b)-len(ipv6Header)))
	// Next protocol
	b[6] = 58
	// Source IP
	copy(b[8:24], src)
	copy(b[24:40], src)

	// ICMPv6 checksum.
	checksum := &onesComplement{}
	checksum.Add(b[8:40])
	checksum.Add(b[4:6])
	checksum.Add(b[6:7])
	checksum.Add(b[40:])
	binary.BigEndian.PutUint16(b[42:44], checksum.Sum())

	return &tuntap.Packet{Protocol: 0x86dd, Packet: b}
}
