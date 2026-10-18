package acceptance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

type apiFixture struct {
	root, work, cfg string
	server          *http.Server
	reads           atomic.Int32
	invalid         atomic.Bool
}

func startAPIFixture(body string, status int) (*apiFixture, error) {
	if status == 0 {
		status = 200
	}
	return startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		if isMergeMethodsFixtureRead(r) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/repos/owner/current/labels" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[]`)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/repos/owner/current/topics" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"names":[]}`)
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != "/repos/owner/current" {
			http.Error(w, "unexpected fixture request", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}, true)
}

func startHTTPFixture(handler http.HandlerFunc, readOnly bool) (*apiFixture, error) {
	// A short /tmp path stays within Unix socket path limits on both macOS and Linux.
	root, err := os.MkdirTemp("/tmp", "ghs-accept-")
	if err != nil {
		return nil, err
	}
	fixture := &apiFixture{root: root, work: filepath.Join(root, "repo"), cfg: filepath.Join(root, "config")}
	ready := false
	defer func() {
		if !ready {
			fixture.close()
		}
	}()
	for _, dir := range []string{fixture.work, fixture.cfg} {
		if err := os.Mkdir(dir, 0700); err != nil {
			return nil, err
		}
	}
	socket := filepath.Join(root, "api.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, err
	}
	fixture.server = &http.Server{ReadHeaderTimeout: 5 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixture.reads.Add(1)
		if readOnly && !isMergeMethodsFixtureRead(r) && (r.Method != http.MethodGet || (r.URL.Path != "/repos/owner/current" && r.URL.Path != "/repos/owner/current/topics" && r.URL.Path != "/repos/owner/current/labels")) {
			fixture.invalid.Store(true)
			http.Error(w, "unexpected fixture request", 400)
			return
		}
		handler(w, r)
	})}
	go func() { _ = fixture.server.Serve(listener) }()
	if err := os.WriteFile(filepath.Join(fixture.cfg, "config.yml"), []byte("http_unix_socket: "+socket+"\n"), 0600); err != nil {
		return nil, err
	}
	ready = true
	return fixture, nil
}

func (fixture *apiFixture) close() {
	if fixture.server != nil {
		_ = fixture.server.Close()
	}
	_ = os.RemoveAll(fixture.root)
}

func (fixture *apiFixture) checkRequests(noRead bool) error {
	if fixture.invalid.Load() {
		return fmt.Errorf("request was not a read of the current repository")
	}
	if noRead {
		if fixture.reads.Load() != 0 {
			return fmt.Errorf("unexpected API request")
		}
	} else if fixture.reads.Load() == 0 {
		return fmt.Errorf("CLI did not use the local API fixture")
	}
	return nil
}

const mergeMethodsFixtureResponse = `{"data":{"repository":{"mergeCommitAllowed":false,"squashMergeAllowed":false,"rebaseMergeAllowed":false}}}`

// Only this fixed query to the fixture repository is read-only, never arbitrary POSTs.
func isMergeMethodsFixtureRead(r *http.Request) bool {
	if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
		return false
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	var query struct {
		Query     string            `json:"query"`
		Variables map[string]string `json:"variables"`
	}
	if err := json.Unmarshal(body, &query); err != nil {
		return false
	}
	expected := `query($owner: String!, $name: String!) { repository(owner: $owner, name: $name) { mergeCommitAllowed squashMergeAllowed rebaseMergeAllowed } }`
	return strings.Join(strings.Fields(query.Query), " ") == expected && len(query.Variables) == 2 && query.Variables["owner"] == "owner" && query.Variables["name"] == "current"
}
