// A fixed-upstream loopback relay keeps OpenCode outbound traffic inside its Seatbelt policy.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type openCodeRelay struct {
	listener   net.Listener
	upstream   string
	prefix     string
	mu         sync.Mutex
	closed     bool
	active     map[net.Conn]struct{}
	acceptDone chan struct{}
	workers    sync.WaitGroup
	closeOnce  sync.Once
	closeErr   error
	acceptErr  error
	ctx        context.Context
	cancel     context.CancelFunc
}

func startOpenCodeRelay(modelURL string) (*openCodeRelay, error) {
	endpoint, upstream, err := parseOpenCodeRelayEndpoint(modelURL)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp4", "127.0.0.1:0")
	if err != nil {
		cancel()
		return nil, fmt.Errorf("listen for OpenCode model relay: %w", err)
	}
	localURL := &url.URL{Scheme: "http", Host: listener.Addr().String(), Path: endpoint.Path, RawPath: endpoint.RawPath}
	relay := &openCodeRelay{listener: listener, upstream: upstream, prefix: localURL.String(), active: make(map[net.Conn]struct{}), acceptDone: make(chan struct{}), ctx: ctx, cancel: cancel}
	go relay.accept()
	return relay, nil
}

func parseOpenCodeRelayEndpoint(modelURL string) (*url.URL, string, error) {
	endpoint, err := url.Parse(modelURL)
	if err != nil || endpoint.Scheme != "http" || endpoint.Opaque != "" || endpoint.User != nil || endpoint.Host == "" || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" || endpoint.RawFragment != "" || strings.Contains(modelURL, "#") {
		return nil, "", fmt.Errorf("OpenCode relay endpoint %q must be an HTTP URL without credentials, query, or fragment", modelURL)
	}
	address := net.ParseIP(endpoint.Hostname())
	port, portErr := strconv.Atoi(endpoint.Port())
	if !endpoint.IsAbs() || address == nil || endpoint.Port() == "" || portErr != nil || port < 1 || port > 65535 {
		return nil, "", fmt.Errorf("OpenCode relay endpoint %q must use a numeric IP address and explicit valid port", modelURL)
	}
	return endpoint, net.JoinHostPort(address.String(), strconv.Itoa(port)), nil
}

func (r *openCodeRelay) accept() {
	defer close(r.acceptDone)
	for {
		connection, err := r.listener.Accept()
		if err != nil {
			r.mu.Lock()
			if !r.closed {
				r.acceptErr = fmt.Errorf("accept OpenCode relay connection: %w", err)
			}
			r.mu.Unlock()
			return
		}
		r.mu.Lock()
		if r.closed {
			r.mu.Unlock()
			_ = connection.Close()
			return
		}
		r.active[connection] = struct{}{}
		r.workers.Add(1)
		r.mu.Unlock()
		go r.forward(connection)
	}
}

func (r *openCodeRelay) forward(client net.Conn) {
	defer r.workers.Done()
	defer func() { _ = client.Close() }()
	defer r.remove(client)
	dialer := net.Dialer{Timeout: 5 * time.Second}
	upstream, err := dialer.DialContext(r.ctx, "tcp", r.upstream)
	if err != nil {
		return
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		_ = upstream.Close()
		return
	}
	r.active[upstream] = struct{}{}
	r.mu.Unlock()
	defer r.remove(upstream)
	defer func() { _ = upstream.Close() }()
	copied := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(upstream, client)
		if halfCloser, ok := upstream.(interface{ CloseWrite() error }); ok {
			_ = halfCloser.CloseWrite()
		}
		copied <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(client, upstream)
		if halfCloser, ok := client.(interface{ CloseWrite() error }); ok {
			_ = halfCloser.CloseWrite()
		}
		copied <- struct{}{}
	}()
	<-copied
	<-copied
}

func (r *openCodeRelay) remove(connection net.Conn) {
	r.mu.Lock()
	delete(r.active, connection)
	r.mu.Unlock()
}

func (r *openCodeRelay) Close() error {
	r.closeOnce.Do(func() {
		r.cancel()
		r.mu.Lock()
		r.closed = true
		r.mu.Unlock()
		if err := r.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			r.closeErr = fmt.Errorf("close OpenCode relay listener: %w", err)
		}
		<-r.acceptDone
		r.mu.Lock()
		r.closeErr = errors.Join(r.closeErr, r.acceptErr)
		connections := make([]net.Conn, 0, len(r.active))
		for connection := range r.active {
			connections = append(connections, connection)
		}
		r.mu.Unlock()
		for _, connection := range connections {
			_ = connection.Close()
		}
		r.workers.Wait()
	})
	return r.closeErr
}

func (r *openCodeRelay) URL() string {
	return r.prefix
}

func startOpenCodeRelayForJobs(jobs []Job) (*openCodeRelay, error) {
	endpoint := ""
	for _, job := range jobs {
		if job.Target.Harness != string(HarnessOpenCode) {
			continue
		}
		provider, _, _ := strings.Cut(openCodeProviderModel(job.Target.Model), "/")
		if provider != "amdbeast" {
			continue
		}
		candidate := openCodeModelBaseURL(job.Target)
		if endpoint != "" && candidate != endpoint {
			return nil, fmt.Errorf("one benchmark run cannot use multiple OpenCode model endpoints")
		}
		endpoint = candidate
	}
	if endpoint == "" {
		// A plan without local AMD Beast jobs validly has no relay to own.
		//nolint:nilnil // A missing relay is the valid absence result for this plan.
		return nil, nil
	}
	return startOpenCodeRelay(endpoint)
}

func prepareOpenCodeRunRelay(jobs []Job) (string, openCodeModelLimits, func() error, error) {
	relay, err := startOpenCodeRelayForJobs(jobs)
	if err != nil {
		return "", openCodeModelLimits{}, nil, err
	}
	if relay == nil {
		return "", openCodeModelLimits{}, func() error { return nil }, nil
	}
	model, err := selectedOpenCodeRelayModel(jobs)
	if err != nil {
		return "", openCodeModelLimits{}, relay.Close, errors.Join(fmt.Errorf("select AMD Beast model for llama.cpp props discovery: %w", err), relay.Close())
	}
	limits, err := relay.discoverModelLimits(context.Background(), model)
	if err != nil {
		return "", openCodeModelLimits{}, relay.Close, errors.Join(fmt.Errorf("discover AMD Beast model limits: %w", err), relay.Close())
	}
	return relay.URL(), limits, relay.Close, nil
}

const openCodeModelOutputReserve = 8192

type openCodeModelLimits struct {
	Context int
	Output  int
}

type openCodeServerProps struct {
	ModelAlias                string `json:"model_alias"`
	DefaultGenerationSettings struct {
		Context int `json:"n_ctx"`
		Params  struct {
			Predict int `json:"n_predict"`
		} `json:"params"`
	} `json:"default_generation_settings"`
}

func (r *openCodeRelay) PropsURL() string {
	return (&url.URL{Scheme: "http", Host: r.listener.Addr().String(), Path: "/props"}).String()
}

func (r *openCodeRelay) discoverModelLimits(ctx context.Context, model string) (openCodeModelLimits, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, r.PropsURL(), nil)
	if err != nil {
		return openCodeModelLimits{}, fmt.Errorf("create llama.cpp props request: %w", err)
	}
	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		return openCodeModelLimits{}, fmt.Errorf("request llama.cpp props through OpenCode relay: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return openCodeModelLimits{}, fmt.Errorf("request llama.cpp props through OpenCode relay: unexpected status %s", response.Status)
	}
	var props openCodeServerProps
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&props); err != nil {
		return openCodeModelLimits{}, fmt.Errorf("decode llama.cpp props through OpenCode relay: %w", err)
	}
	if props.ModelAlias != model {
		return openCodeModelLimits{}, fmt.Errorf("llama.cpp props model alias %q does not match selected AMD Beast model %q", props.ModelAlias, model)
	}
	if props.DefaultGenerationSettings.Context <= 0 {
		return openCodeModelLimits{}, fmt.Errorf("llama.cpp props model %q has invalid context limit %d", model, props.DefaultGenerationSettings.Context)
	}
	output := openCodeModelOutputReserve
	if props.DefaultGenerationSettings.Params.Predict > 0 && props.DefaultGenerationSettings.Params.Predict < output {
		output = props.DefaultGenerationSettings.Params.Predict
	}
	if props.DefaultGenerationSettings.Context <= output {
		return openCodeModelLimits{}, fmt.Errorf("llama.cpp props model %q context limit %d does not leave room for the %d-token output reserve", model, props.DefaultGenerationSettings.Context, output)
	}
	return openCodeModelLimits{Context: props.DefaultGenerationSettings.Context, Output: output}, nil
}

func selectedOpenCodeRelayModel(jobs []Job) (string, error) {
	model := ""
	for _, job := range jobs {
		if job.Target.Harness != string(HarnessOpenCode) {
			continue
		}
		provider, candidate, found := strings.Cut(openCodeProviderModel(job.Target.Model), "/")
		if provider != "amdbeast" || !found || candidate == "" {
			continue
		}
		if model != "" && candidate != model {
			return "", fmt.Errorf("one benchmark run cannot use multiple AMD Beast models for one llama.cpp props discovery")
		}
		model = candidate
	}
	return model, nil
}
