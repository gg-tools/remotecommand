package server

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

const (
	errorNotFound   = iota
	errorNotAllowed = iota
)

type PTYHandler interface {
	Write(data []byte) (int, error)
	Refresh()
}

// SessionTemplateModel used for templating
type AASessionTemplateModel struct {
	SessionID string
	Salt      string
	WSPath    string
}

// TTYServerConfig is used to configure the tty server before it is started
type TTYServerConfig struct {
	FrontListenAddress string
	FrontendPath       string
	PTY                PTYHandler
	SessionID          string
}

// TTYServer represents the instance of a tty server
type TTYServer struct {
	httpServer *http.Server
	config     TTYServerConfig
	session    *ttyShareSession
}

// NewTTYServer creates a new instance
func NewTTYServer(config TTYServerConfig) (server *TTYServer) {
	server = &TTYServer{
		config: config,
	}
	server.httpServer = &http.Server{
		Addr: config.FrontListenAddress,
	}
	routesHandler := mux.NewRouter()

	installHandlers := func(session string) {
		routesHandler.HandleFunc(fmt.Sprintf("/s/%s/ws", session), func(w http.ResponseWriter, r *http.Request) {
			server.handleWebsocket(w, r)
		})
	}

	// Install the same routes on both the /local/ and /<SessionID>/. The session ID is received
	// from the tty-proxy server, if a public session is involved.
	installHandlers("local")
	installHandlers(config.SessionID)

	server.httpServer.Handler = routesHandler
	server.session = newTTYShareSession(config.PTY)

	return server
}

func (server *TTYServer) handleWebsocket(w http.ResponseWriter, r *http.Request) {
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
		log.Println("Cannot create the WS connection: ", err.Error())
		return
	}

	// On a new connection, ask for a refresh/redraw of the terminal app
	server.config.PTY.Refresh()
	server.session.HandleWSConnection(conn)
}

func (server *TTYServer) Run() (err error) {
	err = server.httpServer.ListenAndServe()
	log.Println("Server finished")
	return
}

func (server *TTYServer) Write(buff []byte) (written int, err error) {
	return server.session.Write(buff)
}

func (server *TTYServer) WindowSize(cols, rows int) (err error) {
	return server.session.WindowSize(cols, rows)
}

func (server *TTYServer) Stop() error {
	log.Println("Stopping the server")
	return server.httpServer.Close()
}
