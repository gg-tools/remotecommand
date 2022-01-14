package http

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func Serve(bindAddr string) error {
	wsShell := NewWSShell()
	m := mux.NewRouter()
	m.HandleFunc(fmt.Sprintf("/s/local/ws"), wsShell.Shell)
	if err := http.ListenAndServe(bindAddr, m); err != nil {
		log.Println("serve http failed", err)
		return err
	}
	return nil
}
