package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gg-tools/remotecommand/internal"
	"github.com/gg-tools/remotecommand/internal/http"
)

func main() {
	listenAddress := flag.String("listen", ":8022", "tty-server address")
	flag.Parse()

	// tty-share works as a server, from here on
	if !internal.IsStdinTerminal() {
		fmt.Printf("Input not a tty\n")
		os.Exit(1)
	}

	_ = http.Serve(*listenAddress)
}
