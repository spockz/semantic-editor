// Package gateway handles request submission through a client.
package gateway

import (
	"errors"

	"example.com/identityservice/request"
	"example.com/identityservice/response"
)

// ResponseClient sends requests and receives responses.
type ResponseClient interface {
	Send(request.Request) (response.Response, error)
}

// Handler manages request submission.
type Handler struct {
	client ResponseClient
}

// Submit sends a request through the client and validates the response.
func (h Handler) Submit(req request.Request) (response.Response, error) {
	if err := req.Validate(); err != nil {
		return response.Response{}, err
	}
	resp, err := h.client.Send(req)
	if err != nil {
		return response.Response{}, err
	}
	if resp.ID == "" {
		return response.Response{}, errors.New("response missing own identity")
	}
	return resp, nil
}
