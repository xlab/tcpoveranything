package main

import "encoding/binary"

func oneAdd(bs ...[]byte) uint16 {
	sum := uint64(0)

	for _, b := range bs {
		for i := 0; i < (len(b) - 1); i += 2 {
			sum += uint64(b[i])<<8 + uint64(b[i+1])
		}
		if len(b)%2 == 1 {
			sum += uint64(b[len(b)-1]) << 8
		}
	}

	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return uint16(sum)
}

func addChecksum(checksum []byte, diff uint16) {
	sum := uint64(^binary.BigEndian.Uint16(checksum))
	sum += uint64(diff)

	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	binary.BigEndian.PutUint16(checksum, ^uint16(sum))
}

func subChecksum(checksum []byte, diff uint16) {
	sum := uint64(^binary.BigEndian.Uint16(checksum))
	sum += uint64(^diff)

	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	binary.BigEndian.PutUint16(checksum, ^uint16(sum))
}
