package gateway

import (
	"errors"
	"example.com/identityservice/request"
	"example.com/identityservice/response"
	"testing"
)

type sentinelClient struct{}

func (sentinelClient) Send(request.Request) (response.Response, error) {
	return response.Response{}, nil
}

type fixedResponseClient struct{ response response.Response }

func (c fixedResponseClient) Send(request.Request) (response.Response, error) {
	return c.response, nil
}

func TestSubmitPreservesSentinelThroughWrapping(t *testing.T) {
	_, err := (Handler{client: sentinelClient{}}).Submit(request.Request{Kind: "audit"})
	if !errors.Is(err, request.ErrInvalidRequest) {
		t.Fatalf("Submit() error = %v, want wrapped ErrInvalidRequest", err)
	}
}

func TestSubmitChecksResponseOwnID(t *testing.T) {
	input := request.Request{RequestID: "req-hidden"}
	missingOwnID := Handler{client: fixedResponseClient{response: response.Response{RequestID: "req-hidden"}}}
	if _, err := missingOwnID.Submit(input); err == nil {
		t.Fatal("Submit() accepted a response with only RequestID and no response ID")
	}
	validOwnID := Handler{client: fixedResponseClient{response: response.Response{ID: "resp-hidden"}}}
	if got, err := validOwnID.Submit(input); err != nil || got.ID != "resp-hidden" {
		t.Fatalf("Submit() = %#v, %v; want response with its own ID", got, err)
	}
}
