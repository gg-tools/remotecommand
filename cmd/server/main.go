package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/gg-tools/remotecommand/internal"
	"github.com/gg-tools/remotecommand/internal/http"
	"github.com/gg-tools/remotecommand/internal/tty"
)

func main() {
	commandName := "bash"
	commandArgs := ""
	listenAddress := flag.String("listen", ":8000", "tty-server address")
	flag.Parse()

	// tty-share works as a server, from here on
	if !internal.IsStdinTerminal() {
		fmt.Printf("Input not a tty\n")
		os.Exit(1)
	}

	ptyMaster := internal.PtyMasterNew()
	envVars := os.Environ()
	err := ptyMaster.Start(commandName, strings.Fields(commandArgs), envVars)
	if err != nil {
		log.Println("Cannot start the %s command: %s", commandName, err.Error())
		return
	}

	fmt.Printf("local session: ws://%s/s/local/ws\n", *listenAddress)
	stopPtyAndRestore := func() {
		ptyMaster.Stop()
		ptyMaster.Restore()
	}

	ptyMaster.MakeRaw()
	defer stopPtyAndRestore()

	pty := ptyMaster
	s := createServer(*listenAddress, pty)
	if cols, rows, e := ptyMaster.GetWinSize(); e == nil {
		s.WindowSize(cols, rows)
	}

	ptyMaster.SetWinChangeCB(func(cols, rows int) {
		log.Print("New window size: %dx%d", cols, rows)
		s.WindowSize(cols, rows)
	})

	go func() {
		err := s.Run()
		if err != nil {
			stopPtyAndRestore()
			log.Println("Server finished: %s", err.Error())
		}
	}()

	go func() {
		_, err := io.Copy(s, ptyMaster)
		if err != nil {
			stopPtyAndRestore()
		}
	}()

	go func() {
		_, err := io.Copy(ptyMaster, os.Stdin)
		if err != nil {
			stopPtyAndRestore()
		}
	}()

	ptyMaster.Wait()
	fmt.Printf("tty-share finished\n\n\r")
	s.Stop()
}

func createServer(frontListenAddress string, pty tty.PTYHandler) *http.TTYServer {
	config := http.TTYServerConfig{
		FrontListenAddress: frontListenAddress,
		PTY:                pty,
	}

	s := http.NewTTYServer(config)
	return s
}
