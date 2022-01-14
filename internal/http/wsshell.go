package http

import (
	"github.com/gg-tools/remotecommand/internal"
	"github.com/gg-tools/remotecommand/internal/tty"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type WSShell struct {
}

func NewWSShell() *WSShell {
	return &WSShell{}
}

func (s *WSShell) Shell(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println("cannot create the WS connection: ", err.Error())
		return
	}

	// session
	sess, err := createSession()
	if err != nil {
		log.Println("cannot create session: ", err.Error())
		return
	}
	sess.setup()

	// On a new connection, ask for a refresh/redraw of the terminal app
	sess.pty.Refresh()
	sess.session.HandleWSConnection(conn)
}

type session struct {
	pty     *internal.PtyMaster
	session *tty.TTYShareSession
}

func (s *session) Write(buff []byte) (written int, err error) {
	return s.session.Write(buff)
}

func (s *session) setup() {
	stopPtyAndRestore := func() {
		s.pty.Stop()
		s.pty.Restore()
	}
	// defer stopPtyAndRestore()

	if cols, rows, e := s.pty.GetWinSize(); e == nil {
		s.session.WindowSize(cols, rows)
	}

	s.pty.SetWinChangeCB(func(cols, rows int) {
		log.Print("new window size: %dx%d", cols, rows)
		s.session.WindowSize(cols, rows)
	})

	go func() {
		_, err := io.Copy(s, s.pty)
		if err != nil {
			stopPtyAndRestore()
		}
	}()

	go func() {
		_, err := io.Copy(s.pty, os.Stdin)
		if err != nil {
			stopPtyAndRestore()
		}
	}()

}

func createSession() (*session, error) {
	commandName := "bash"
	commandArgs := ""

	ptyMaster := internal.PtyMasterNew()
	envVars := os.Environ()
	err := ptyMaster.Start(commandName, strings.Fields(commandArgs), envVars)
	if err != nil {
		log.Println("cannot start the %s command: %s", commandName, err.Error())
		return nil, err
	}
	ptyMaster.MakeRaw()

	// stopPtyAndRestore := func() {
	//	ptyMaster.Stop()
	//	ptyMaster.Restore()
	// }
	// defer stopPtyAndRestore()

	pty := ptyMaster
	return &session{
		pty:     pty,
		session: tty.NewTTYShareSession(pty),
	}, nil
}
