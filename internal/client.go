package internal

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/gg-tools/remotecommand/internal/tty"
	"github.com/gorilla/websocket"
	"github.com/moby/term"
	"log"
)

type ttyShareClient struct {
	url          string
	wsConn       *websocket.Conn
	detachKeys   string
	wcChan       chan os.Signal
	ioFlagAtomic uint32 // used with atomic
	winSizes     struct {
		thisW   uint16
		thisH   uint16
		remoteW uint16
		remoteH uint16
	}
	winSizesMutex sync.Mutex
}

func NewTtyShareClient(url string, detachKeys string) *ttyShareClient {
	return &ttyShareClient{
		url:          url,
		wsConn:       nil,
		detachKeys:   detachKeys,
		wcChan:       make(chan os.Signal, 1),
		ioFlagAtomic: 1,
	}
}

func clearScreen() {
	fmt.Fprintf(os.Stdout, "\033[H\033[2J")
}

type keyListener struct {
	wrappedReader io.Reader
	ioFlagAtomicP *uint32
}

func (kl *keyListener) Read(data []byte) (n int, err error) {
	n, err = kl.wrappedReader.Read(data)
	if _, ok := err.(term.EscapeError); ok {
		log.Println("Escape code detected.")
	}

	// If we are not supposed to do any IO, then return 0 bytes read. This happens the local
	// window is smaller than the remote one
	if atomic.LoadUint32(kl.ioFlagAtomicP) == 0 {
		return 0, err
	}

	return
}

func (c *ttyShareClient) updateAndDecideStdoutMuted() {
	log.Println("This window: %dx%d. Remote window: %dx%d", c.winSizes.thisW, c.winSizes.thisH, c.winSizes.remoteW, c.winSizes.remoteH)

	if c.winSizes.thisH < c.winSizes.remoteH || c.winSizes.thisW < c.winSizes.remoteW {
		atomic.StoreUint32(&c.ioFlagAtomic, 0)
		clearScreen()
		messageFormat := "\n\rYour window is smaller than the remote window. Please resize or press <C-o C-c> to detach.\n\r\tRemote window: %dx%d \n\r\tYour window:   %dx%d \n\r"
		fmt.Printf(messageFormat, c.winSizes.remoteW, c.winSizes.remoteH, c.winSizes.thisW, c.winSizes.thisH)
	} else {
		if atomic.LoadUint32(&c.ioFlagAtomic) == 0 { // clear the screen when changing back to "write"
			// TODO: notify the remote side to "refresh" the content.
			clearScreen()
		}
		atomic.StoreUint32(&c.ioFlagAtomic, 1)
	}
}

func (c *ttyShareClient) updateThisWinSize() {
	size, err := term.GetWinsize(os.Stdin.Fd())
	if err == nil {
		c.winSizesMutex.Lock()
		c.winSizes.thisW = size.Width
		c.winSizes.thisH = size.Height
		c.winSizesMutex.Unlock()
	}
}

func (c *ttyShareClient) Run() (err error) {
	log.Println("Connecting as a client to %s ..", c.url)

	c.wsConn, _, err = websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return
	}

	detachBytes, err := term.ToBytes(c.detachKeys)
	if err != nil {
		log.Println("Invalid dettaching keys: %s", c.detachKeys)
		return
	}

	state, err := term.MakeRaw(os.Stdin.Fd())
	defer term.RestoreTerminal(os.Stdin.Fd(), state)
	clearScreen()

	protoWS := tty.NewTTYProtocolWSLocked(c.wsConn)

	monitorWinChanges := func() {
		// start monitoring the size of the terminal
		signal.Notify(c.wcChan, syscall.SIGWINCH)

		for {
			select {
			case <-c.wcChan:
				c.updateThisWinSize()
				c.updateAndDecideStdoutMuted()
				protoWS.SetWinSize(int(c.winSizes.thisW), int(c.winSizes.thisH))
			}
		}
	}

	readLoop := func() {

		var err error
		for {
			err = protoWS.ReadAndHandle(
				// onWrite
				func(data []byte) {
					if atomic.LoadUint32(&c.ioFlagAtomic) != 0 {
						os.Stdout.Write(data)
					}
				},
				// onWindowSize
				func(cols, rows int) {
					c.winSizesMutex.Lock()
					c.winSizes.remoteW = uint16(cols)
					c.winSizes.remoteH = uint16(rows)
					c.winSizesMutex.Unlock()
					c.updateThisWinSize()
					c.updateAndDecideStdoutMuted()
				},
			)

			if err != nil {
				log.Println("Error parsing remote message: %s", err.Error())
				if err == io.EOF {
					// Remote WS connection closed
					return
				}
			}
		}
	}

	writeLoop := func() {
		kl := &keyListener{
			wrappedReader: term.NewEscapeProxy(os.Stdin, detachBytes),
			ioFlagAtomicP: &c.ioFlagAtomic,
		}
		_, err := io.Copy(protoWS, kl)

		if err != nil {
			log.Println("Connection closed: %s", err.Error())
			c.Stop()
			return
		}
	}

	go monitorWinChanges()
	go writeLoop()
	readLoop()

	clearScreen()
	return
}

func (c *ttyShareClient) Stop() {
	c.wsConn.Close()
	signal.Stop(c.wcChan)
}
