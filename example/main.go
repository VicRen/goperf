package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"vic.ren/goperf"
)

var (
	s = flag.String("s", "", "s N, server, listen on port N (all interfaces)")
	c = flag.String("c", "", "c host:port, client, make connection to host:port")
)

func main() {
	flag.Parse()

	if *s == "" {
		runClient()
	} else {
		runServer()
	}
}

func runServer() {
	output := goperf.NewReceivingOutput(os.Stdout, true)
	/* Let's prepare a address at any address at port s */
	ServerAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0"+":"+*s)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	/* Now listen at selected port */
	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	defer ServerConn.Close()

	if err := goperf.RunUDPServer(ServerConn, output, 1000000, -1); err != nil {
		panic(err)
	}

	fmt.Println("Last Data:", output.FetchLastData())
}

func runClient() {
	output := goperf.NewSendingOutput(os.Stdout, false)
	ServerAddr, err := net.ResolveUDPAddr("udp", *c)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	Conn, err := net.DialUDP("udp", nil, ServerAddr)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	defer Conn.Close()

	if err := goperf.RunUDPClient(Conn, output, 100, 1000, -1, 1000000); err != nil {
		panic(err)
	}

	fmt.Println("Last Data:", output.FetchLastData())
}
