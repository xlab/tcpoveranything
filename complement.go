package main

type onesComplement struct {
	sum uint64
}

func (o *onesComplement) Add(b []byte) {
	for len(b) > 1 {
		o.sum += uint64(b[0])<<8 + uint64(b[1])
		b = b[2:]
	}
	if len(b) > 0 {
		o.sum += uint64(b[0])
	}
}

func (o *onesComplement) Sub(b []byte) {
	for len(b) > 1 {
		o.sum += ^(uint64(b[0])<<8 + uint64(b[1]))
		b = b[2:]
	}
	if len(b) > 0 {
		o.sum += ^uint64(b[0])
	}
}

func (o *onesComplement) Sum() uint16 {
	for o.sum >> 16 != 0 {
		o.sum = (o.sum & 0xffff) + (o.sum >> 16)
	}
	return ^uint16(o.sum)
}
