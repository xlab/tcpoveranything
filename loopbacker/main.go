package main

import (
	"fmt"
	"net"
	"io/ioutil"
	"io"
	"bufio"
)

func panicif(err error) {
	if err != nil {
		panic(err)
	}
}

func server() {
	ln, err := net.Listen("tcp", ":5656")
	panicif(err)

	for {
		sock, err := ln.Accept()
		panicif(err)
		sock.Write([]byte("Hi there!\n"))
		sock.Close()
	}
}

func client(lport int, dest string) {
	laddr, err := net.ResolveTCPAddr("tcp6", fmt.Sprintf("[::]:%d", lport))
	panicif(err)
	raddr, err := net.ResolveTCPAddr("tcp6", dest)
	panicif(err)

	sock, err := net.DialTCP("tcp6", laddr, raddr)
	panicif(err)
	b, err := ioutil.ReadAll(sock)
	panicif(err)
	fmt.Println("Client:", string(b))
}

func register(port int) (string, io.Reader, io.Writer)  {
	sock, err := net.Dial("tcp", "localhost:1943")
	panicif(err)

	r := bufio.NewReader(sock)
	_, _, err = r.ReadLine()
	panicif(err)
	_, err = fmt.Fprintln(sock, port)
	panicif(err)
	addr, _, err := r.ReadLine()
	panicif(err)
	return string(addr), r, sock
}

func main() {
	go server()
	_, ar, aw := register(5656)
	addr, br, bw := register(5657)
	go io.Copy(bw, ar)
	go io.Copy(aw, br)
	client(5657, addr)
}
