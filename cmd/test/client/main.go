package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var (
	remote = flag.String("r", "172.25.108.254:8022", "Address of the golocproxy server")
)

func main() {
	flag.Parse()

	log.Println("client starting: ", *remote)
	for {
		connectServer()
		time.Sleep(10 * time.Second) // retry after 10s
	}
}

func connectServer() {
	conn, err := net.DialTimeout("tcp", *remote, 5*time.Second)
	if err != nil {
		log.Println("CAN'T CONNECT:", *remote, " err:", err)
		return
	}

	go func() {
		for {
			_, err := io.Copy(os.Stdout, conn)
			if err != nil {
				log.Println("fail to receive: ", err)
				continue
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		// Read the keyboad input.
		input, err := reader.ReadString('\n')
		input = strings.Replace(input, "\r", "", -1)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		// Handle the execution of the input.
		if _, err := conn.Write([]byte(input + "\n")); err != nil {
			log.Println("fail to send line: ", err)
			return
		}
	}
}
