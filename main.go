package main

import (
	"github.com/moby/term"
	"log"
	"os"
)

func main() {
	fd := os.Stdin.Fd()

	if term.IsTerminal(fd) {
		ws, err := term.GetWinsize(fd)
		if err != nil {
			log.Fatalf("term.GetWinsize: %s", err)
		}
		log.Printf("%d:%d\n", ws.Height, ws.Width)
	}
}