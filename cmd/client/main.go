package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/gg-tools/remotecommand/internal"
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("Usage: client [service address]")
		return
	}

	connectURL := args[0]
	client := internal.NewTtyShareClient(connectURL, "ctrl-c")

	err := client.Run()
	if err != nil {
		log.Println("Cannot connect to the remote session. Make sure the URL points to a valid tty-share session.")
	}
	log.Println("tty-share disconnected")
	return
}
