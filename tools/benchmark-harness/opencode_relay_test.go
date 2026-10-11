// Relay tests prove fixed endpoint forwarding, endpoint validation, and bounded shutdown.

package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenCodeRelayForwardsToConfiguredEndpoint(t *testing.T) {
	upstream, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := upstream.Close(); closeErr != nil {
			t.Errorf("close upstream listener: %v", closeErr)
		}
	}()
	endpoint := "http://" + upstream.Addr().String() + "/v1"
	relay, err := startOpenCodeRelay(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := relay.Close(); closeErr != nil {
			t.Errorf("close relay: %v", closeErr)
		}
	}()
	if !strings.HasPrefix(relay.URL(), "http://127.0.0.1:") || !strings.HasSuffix(relay.URL(), "/v1") {
		t.Fatalf("relay URL = %q", relay.URL())
	}
	serverDone := make(chan error, 1)
	go func() {
		connection, acceptErr := upstream.Accept()
		if acceptErr != nil {
			serverDone <- acceptErr
			return
		}
		defer func() { _ = connection.Close() }()
		request, readErr := bufio.NewReader(connection).ReadString('\n')
		if readErr != nil {
			serverDone <- readErr
			return
		}
		if !strings.HasPrefix(request, "GET /v1/models ") {
			serverDone <- fmt.Errorf("upstream request line = %q", request)
			return
		}
		_, writeErr := io.WriteString(connection, "HTTP/1.1 200 OK\r\nContent-Length: 4\r\nConnection: close\r\n\r\npass")
		serverDone <- writeErr
	}()
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(relay.URL() + "/models")
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if response.StatusCode != http.StatusOK || string(body) != "pass" {
		t.Fatalf("relay response = %d %q", response.StatusCode, body)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
}

func TestParseOpenCodeRelayEndpointFailsClosed(t *testing.T) {
	for _, endpoint := range []string{
		"https://192.168.1.122:1234/v1",
		"http://model.local:1234/v1",
		"http://192.168.1.122/v1",
		"http://192.168.1.122:0/v1",
		"http://192.168.1.122:65536/v1",
		"http://user@192.168.1.122:1234/v1",
		"http://192.168.1.122:1234/v1?key=value",
		"http://192.168.1.122:1234/v1?",
		"http://192.168.1.122:1234/v1#fragment",
		"http://192.168.1.122:1234/v1#",
	} {
		if _, _, err := parseOpenCodeRelayEndpoint(endpoint); err == nil {
			t.Errorf("endpoint %q was accepted", endpoint)
		}
	}
	_, upstream, err := parseOpenCodeRelayEndpoint("http://192.168.1.122:1234/v1")
	if err != nil {
		t.Fatal(err)
	}
	if upstream != "192.168.1.122:1234" {
		t.Fatalf("upstream = %q", upstream)
	}
}

func TestOpenCodeRelayCloseStopsActiveConnections(t *testing.T) {
	upstream, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := upstream.Close(); closeErr != nil {
			t.Errorf("close upstream listener: %v", closeErr)
		}
	}()
	relay, err := startOpenCodeRelay("http://" + upstream.Addr().String() + "/v1")
	if err != nil {
		t.Fatal(err)
	}
	client, err := net.Dial("tcp4", strings.TrimSuffix(strings.TrimPrefix(relay.URL(), "http://"), "/v1"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()
	accepted := make(chan net.Conn, 1)
	go func() {
		connection, acceptErr := upstream.Accept()
		if acceptErr == nil {
			accepted <- connection
		}
	}()
	var serverConn net.Conn
	select {
	case serverConn = <-accepted:
	case <-time.After(time.Second):
		t.Fatal("relay did not connect to its configured upstream")
	}
	defer func() { _ = serverConn.Close() }()
	closed := make(chan error, 1)
	go func() { closed <- relay.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("relay close waited on an active connection")
	}
	if _, err := net.DialTimeout("tcp4", strings.TrimSuffix(strings.TrimPrefix(relay.URL(), "http://"), "/v1"), 100*time.Millisecond); err == nil {
		t.Fatal("relay listener accepted a connection after close")
	}
}

func TestOpenCodeRelayForJobsSharesOneRunEndpoint(t *testing.T) {
	t.Setenv(openCodeQwenBaseURLEnv, "http://127.0.0.1:43210/v1")
	jobs := []Job{
		{Target: Target{Harness: string(HarnessOpenCode), Model: "amdbeast/model-a"}},
		{Target: Target{Harness: string(HarnessOpenCode), Model: "amdbeast/model-b"}},
		{Target: Target{Harness: string(HarnessOpenCode), Model: "openrouter/model-c"}},
	}
	relay, err := startOpenCodeRelayForJobs(jobs)
	if err != nil {
		t.Fatal(err)
	}
	if relay == nil {
		t.Fatal("AMD Beast jobs did not start a run relay")
	}
	defer func() {
		if closeErr := relay.Close(); closeErr != nil {
			t.Errorf("close relay: %v", closeErr)
		}
	}()
	if !strings.HasPrefix(relay.URL(), "http://127.0.0.1:") || !strings.HasSuffix(relay.URL(), "/v1") {
		t.Fatalf("run relay URL = %q", relay.URL())
	}
	if relay, err := startOpenCodeRelayForJobs([]Job{{Target: Target{Harness: string(HarnessOpenCode), Model: "openrouter/model"}}}); err != nil || relay != nil {
		t.Fatalf("OpenRouter-only jobs relay = %v, error = %v; want no relay", relay, err)
	}
}

func TestOpenCodeRelayHandlesConcurrentClients(t *testing.T) {
	upstream, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := upstream.Close(); closeErr != nil {
			t.Errorf("close upstream listener: %v", closeErr)
		}
	}()
	relay, err := startOpenCodeRelay("http://" + upstream.Addr().String() + "/v1")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := relay.Close(); closeErr != nil {
			t.Errorf("close relay: %v", closeErr)
		}
	}()
	acceptDone := acceptOpenCodeEchoClients(upstream, 2)
	type result struct {
		want string
		got  string
		err  error
	}
	results := make(chan result, 2)
	for _, payload := range []string{"first-client", "second-client"} {
		go func(payload string) {
			got, err := roundTripOpenCodeRelay(t, relay.URL(), payload)
			results <- result{want: payload, got: got, err: err}
		}(payload)
	}
	deadline := time.After(3 * time.Second)
	for range 2 {
		select {
		case got := <-results:
			if got.err != nil || got.got != got.want {
				t.Errorf("relay echo want %q, got %q: %v", got.want, got.got, got.err)
			}
		case <-deadline:
			t.Fatal("concurrent relay clients did not receive their responses")
		}
	}
	select {
	case err := <-acceptDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-deadline:
		t.Fatal("upstream did not accept both relay clients")
	}
}

func acceptOpenCodeEchoClients(listener net.Listener, count int) <-chan error {
	done := make(chan error, 1)
	go func() {
		for range count {
			connection, err := listener.Accept()
			if err != nil {
				done <- err
				return
			}
			go func() {
				defer func() { _ = connection.Close() }()
				_, _ = io.Copy(connection, connection)
			}()
		}
		done <- nil
	}()
	return done
}

func roundTripOpenCodeRelay(t *testing.T, relayURL, payload string) (string, error) {
	t.Helper()
	hostPort := strings.TrimPrefix(relayURL, "http://")
	hostPort, _, _ = strings.Cut(hostPort, "/")
	connection, dialErr := net.DialTimeout("tcp4", hostPort, time.Second)
	if dialErr != nil {
		return "", dialErr
	}
	defer func() { _ = connection.Close() }()
	if _, writeErr := io.WriteString(connection, payload); writeErr != nil {
		return "", writeErr
	}
	if halfCloser, ok := connection.(interface{ CloseWrite() error }); ok {
		if closeErr := halfCloser.CloseWrite(); closeErr != nil {
			return "", closeErr
		}
	}
	body, readErr := io.ReadAll(connection)
	return string(body), readErr
}

func TestPrepareOpenCodeRunRelayDiscoversLlamaCppLimitsOnce(t *testing.T) {
	const model = "Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest"
	propsCalls := 0
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/props" {
			http.NotFound(writer, request)
			return
		}
		propsCalls++
		_, _ = io.WriteString(writer, `{"model_alias":"Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest","default_generation_settings":{"n_ctx":65536,"params":{"n_predict":-1}}}`)
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()
	t.Setenv(openCodeQwenBaseURLEnv, server.URL+"/v1")

	relayURL, limits, closeRelay, err := prepareOpenCodeRunRelay([]Job{{Target: Target{Harness: string(HarnessOpenCode), Model: "amdbeast/" + model}}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := closeRelay(); closeErr != nil {
			t.Errorf("close relay: %v", closeErr)
		}
	}()
	if !strings.HasSuffix(relayURL, "/v1") {
		t.Fatalf("relay URL = %q, want API path", relayURL)
	}
	if limits != (openCodeModelLimits{Context: 65536, Output: openCodeModelOutputReserve}) {
		t.Fatalf("limits = %+v", limits)
	}
	if propsCalls != 1 {
		t.Fatalf("props requests = %d, want 1", propsCalls)
	}
}
