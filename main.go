package main

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"net"
	"strconv"

	"code.google.com/p/tuntap"
)

var port = flag.Uint("port", 1943, "Port for binding requests")


func init() {
	var b [8]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic("Universe ran out of random")
	}
	mrand.Seed(int64(binary.BigEndian.Uint64(b[:])))
}

func main() {
	if err := startDevPump(); err != nil {
		log.Fatalln("Couldn't set up virtual interface:", err)
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		log.Fatalln("Couldn't listen on localhost:1943:", err)
	}

	log.Println("Serving IPs from", tunAddr)
	log.Println("Listening on port", *port)
	go pump()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)

	fmt.Fprintln(conn, tunAddr.IP.String())

	resp, prefix, err := r.ReadLine()
	if err != nil && prefix {
		log.Println("Error getting port number:", err)
		return
	}
	localPort, err := strconv.ParseUint(string(resp), 10, 32)
	if err != nil {
		return
	}

	bind := &binding{
		&net.TCPAddr{tunAddr.IP, int(localPort)},
		&net.TCPAddr{randomIp(), 4242},
		conn,
	}

	log.Println("Adding binding", bind)
	fmt.Fprintln(conn, bind.virtual.String())

	bindChange <- bind
	defer func() {
		bindChange <- &binding{bind.local, bind.virtual, nil}
	}()

	const maxPktLen = 20 * 1024
	var pktlen uint32
	var pkt [unMungeStart + maxPktLen]byte
	for {
		if err = binary.Read(r, binary.BigEndian, &pktlen); err != nil {
			log.Println("Error reading from remote socket:", err)
			return
		}
		if pktlen > maxPktLen {
			log.Println("Remote wants me to read a too big packet, bailing")
			return
		}
		if _, err := io.ReadFull(r, pkt[unMungeStart:unMungeStart+pktlen]); err != nil {
			log.Println("Short read:", err)
			return
		}
		if err = tunDev.WritePacket(unMungePacket(pkt[:unMungeStart+pktlen], bind)); err != nil {
			log.Println("Failed to write packet:", err)
			return
		}
		// TODO: decode, munge, forward to tunDev
	}
}

type binding struct {
	local, virtual *net.TCPAddr
	remote         net.Conn
}

func (b *binding) String() string {
	var sock net.Addr
	if b.remote != nil {
		sock = b.remote.RemoteAddr()
	}
	return fmt.Sprintf("%s -> (%s) -> %s", b.local, b.virtual, sock)
}

var bindChange = make(chan *binding)
var tunDev *tuntap.Interface
var tunAddr = func() *net.IPNet {
	_, addr, err := net.ParseCIDR("fd00::/64")
	if err != nil {
		panic("fd00::/64 doesn't parse")
	}
	if _, err = io.ReadFull(rand.Reader, addr.IP[1:8]); err != nil {
		panic("universe ran out of random")
	}
	addr.IP[15] = 1
	return addr
}()

func randomIp() net.IP {
	ip := net.IP(make([]byte, 16))
	copy(ip, tunAddr.IP)
	if _, err := io.ReadFull(rand.Reader, ip[8:16]); err != nil {
		panic("universe has stopped being random")
	}
	return ip
}

func startDevPump() error {
	var err error
	tunDev, err = tuntap.Open("tunTCP%d", tuntap.DevTun)
	if err != nil {
		return err
	}

	if err := setupTun(tunDev, tunAddr); err != nil {
		tunDev.Close()
		return err
	}

	return nil
}

func pump() {
	pkts := make(chan *tuntap.Packet)
	go func() {
		for {
			pkt, err := tunDev.ReadPacket()
			if err != nil {
				panic("Failed read from tun device")
			}
			pkts <- pkt
		}
	}()

	m := make(map[string]*binding)

	for {
		select {
		case pkt := <-pkts:
			dst, payload, new_sum := mungePacket(pkt)
			if dst != nil {
				b, ok := m[dst.String()]
				if ok {
					binary.BigEndian.PutUint16(payload[12:14], new_sum)
					binary.Write(b.remote, binary.BigEndian, uint32(len(payload)))
					if n, err := b.remote.Write(payload); err != nil || n != len(payload) {
						log.Println("Short write")
						// Triggers teardown in the reader
						b.remote.Close()
					}
				} else {
					tunDev.WritePacket(icmpError(tunAddr.IP, pkt.Packet))
					// TODO: log error if any
				}
			}
		case b := <-bindChange:
			if b.remote == nil {
				log.Println("Removing binding", b)
				delete(m, b.virtual.String())
			} else {
				m[b.virtual.String()] = b
			}
		}
	}
}
