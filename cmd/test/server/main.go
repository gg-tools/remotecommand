package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
)

var (
	port = flag.String("p", "8022", "The Listen port of golocproxy, golocproxy client will access the port.")
)

type OnConnectFunc func(net.Conn)

func main() {
	flag.Parse()

	if nil != listen(*port, onConnect) {
		return
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop)
	<-stop
}

func listen(port string, onConnect OnConnectFunc) error {
	server, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", port))
	if err != nil {
		log.Fatal("CAN'T LISTEN: ", err)
		return err
	}
	log.Println("listen port:", port)
	go func() {
		defer server.Close()
		for {
			conn, err := server.Accept()
			if err != nil {
				log.Println("Can't Accept: ", err)
				continue
			}
			go onConnect(conn)
		}
	}()
	return nil
}

func onConnect(conn net.Conn) {
	log.Println("client connect:", conn.RemoteAddr().String())

	c := exec.Command("bash")
	in, err := c.StdinPipe()
	if err != nil {
		log.Println("fail to create stdin pipe")
		return
	}
	out, err := c.StdoutPipe()
	if err != nil {
		log.Println("fail to create stdout pipe")
		return
	}
	if err := c.Start(); err != nil {
		log.Println("fail to start bash")
		return
	}

	go func() {
		if _, err := io.Copy(in, conn); err != nil {
			log.Println("fail to copy:", err)
			return
		}
	}()

	go func() {
		if _, err := io.Copy(conn, out); err != nil {
			log.Println("fail to copy:", err)
			return
		}
	}()
}
