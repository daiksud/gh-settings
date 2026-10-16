// Command plan-fixture serves the candidate composite's fixed local repository.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
		authorized := r.Method == http.MethodGet && r.Header.Get("Authorization") == "token plan-fixture-read" ||
			r.Method == http.MethodPatch && r.Header.Get("Authorization") == "token apply-fixture-write"
		if !authorized || r.URL.RequestURI() != "/repos/example-owner/settings-fixture" || r.Host != "api.github.com" {
			fmt.Fprintln(log, "unexpected request")
			http.Error(w, "unexpected fixture request", http.StatusBadRequest)
			return
		}
		fmt.Fprintln(log, r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPatch {
			body, err := io.ReadAll(r.Body)
			var payload map[string]any
			if err == nil {
				err = json.Unmarshal(body, &payload)
			}
			if err == nil {
				body, err = json.Marshal(payload)
			}
			if err == nil {
				err = os.WriteFile(filepath.Join(root, "patch.json"), body, 0600)
			}
			if err != nil {
				fmt.Fprintln(log, "invalid PATCH payload")
				http.Error(w, "invalid fixture payload", http.StatusBadRequest)
				return
			}
			if _, err := os.Stat(filepath.Join(root, "write403")); err == nil {
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprintln(w, `{"message":"fixture write denied"}`)
				return
			}
			fmt.Fprintln(w, `{}`)
			return
		}
		if _, err := os.Stat(filepath.Join(root, "read403")); err == nil {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintln(w, `{"message":"fixture read denied"}`)
			return
		}
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
