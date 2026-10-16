// Command plan-fixture serves the candidate composite's fixed local repository.
package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: plan-fixture STATE_DIRECTORY")
	}
	root := os.Args[1]
	log, err := os.OpenFile(filepath.Join(root, "requests"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer log.Close()
	listener, err := net.Listen("unix", filepath.Join(root, "api.sock"))
	if err != nil {
		return err
	}
	defer listener.Close()
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.RequestURI() != "/repos/example-owner/settings-fixture" || r.Host != "api.github.com" || r.Header.Get("Authorization") != "token plan-fixture-read" {
			fmt.Fprintln(log, "unexpected request")
			http.Error(w, "unexpected fixture request", http.StatusBadRequest)
			return
		}
		fmt.Fprintln(log, r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"description":"before <b>bold</b>\nline","homepage":"","has_issues":true,"has_projects":true,"has_wiki":true}`)
	})}
	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stopped)
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	if err := os.WriteFile(filepath.Join(root, "ready"), nil, 0600); err != nil {
		_ = server.Close()
		return err
	}
	select {
	case <-stopped:
		if err := server.Close(); err != nil {
			return err
		}
		err = <-served
	case err = <-served:
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
