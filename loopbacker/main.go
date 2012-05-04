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

var ch = make(chan interface{})

func server() {
	ln, err := net.Listen("tcp", ":33333")
	panicif(err)

	sock, err := ln.Accept()
	panicif(err)
	sock.Write([]byte("Hi there!\n"))
	sock.Close()
	ch<-nil
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
	sock.Close()
	ch<-nil
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
	_, ar, aw := register(33333)
	addr, br, bw := register(22222)
	go io.Copy(bw, ar)
	go io.Copy(aw, br)
	go client(22222, addr)
	<-ch
	<-ch
}
