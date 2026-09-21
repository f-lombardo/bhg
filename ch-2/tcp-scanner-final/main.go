package main

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"time"
)

func worker(i int, host string, ports, results chan int) {
	for p := range ports {
		address := net.JoinHostPort(host, strconv.Itoa(p))
		conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err != nil {
			fmt.Printf("Goroutine %d. Failed to connect to %s\n", i, address)
			results <- 0
			continue
		}
		conn.Close()
		fmt.Printf("Goroutine %d. Connected to %s\n", i, address)
		results <- p
	}
}

func main() {

	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s <host>\n", os.Args[0])
		os.Exit(2)
	}

	host := os.Args[1]

	ports := make(chan int, 100)
	results := make(chan int)
	var openports []int

	nrOfGoroutines := 10
	for i := 0; i < nrOfGoroutines; i++ {
		go worker(i, host, ports, results)
	}

	nrOfPorts := 1024
	go func() {
		for i := 1; i <= nrOfPorts; i++ {
			ports <- i
		}
	}()

	for i := 0; i < nrOfPorts; i++ {
		port := <-results
		if port != 0 {
			openports = append(openports, port)
		}
	}

	close(ports)
	close(results)
	sort.Ints(openports)
	for _, port := range openports {
		fmt.Printf("%d open\n", port)
	}
}
