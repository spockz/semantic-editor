// Package scala owns one-shot Metals apply-edit capture for command-based inline-value actions.
// WHY: Metals does not correlate or await workspace/applyEdit with its command response.
package scala

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/spockz/semantic-editor/internal/lsp"
)

const scalaApplyEditWait = 10 * time.Second

type scalaProcessSession struct {
	client *lsp.Client

	applyMu sync.Mutex
	mu      sync.Mutex
	apply   func(json.RawMessage) (bool, error)
	active  sync.WaitGroup
}

type scalaApplyEditResult struct {
	applied bool
	err     error
}

func (s *scalaProcessSession) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	return s.client.Request(ctx, method, params)
}

func (s *scalaProcessSession) Notify(ctx context.Context, method string, params any) error {
	return s.client.Notify(ctx, method, params)
}

func (s *scalaProcessSession) Close() error { return s.client.Close() }

func (s *scalaProcessSession) RequestWithApplyEdit(ctx context.Context, method string, params any, capture func(json.RawMessage) (bool, error)) (response json.RawMessage, err error) {
	if capture == nil {
		return nil, errors.New("scala apply-edit capture requires a validator")
	}
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	completed := make(chan scalaApplyEditResult, 1)
	var captureMu sync.Mutex
	var captured bool
	var callbackErr error
	s.mu.Lock()
	if s.apply != nil {
		s.mu.Unlock()
		return nil, errors.New("scala apply-edit capture is already active")
	}
	s.apply = func(raw json.RawMessage) (bool, error) {
		captureMu.Lock()
		defer captureMu.Unlock()
		if captured {
			callbackErr = errors.Join(callbackErr, errors.New("metals sent more than one workspace/applyEdit request"))
			return false, callbackErr
		}
		captured = true
		applied, captureErr := capture(raw)
		if captureErr != nil {
			callbackErr = errors.Join(callbackErr, captureErr)
		} else if !applied {
			callbackErr = errors.Join(callbackErr, errors.New("metals workspace/applyEdit was declined"))
		}
		select {
		case completed <- scalaApplyEditResult{applied: applied, err: captureErr}:
		default:
		}
		return applied, captureErr
	}
	s.mu.Unlock()
	defer func() {
		s.endApplyEditCapture()
		captureMu.Lock()
		defer captureMu.Unlock()
		if callbackErr != nil {
			response = nil
			err = errors.Join(err, fmt.Errorf("capture Metals workspace/applyEdit: %w", callbackErr))
		}
	}()

	response, err = s.client.Request(ctx, method, params)
	if err != nil {
		return nil, err
	}
	timer := time.NewTimer(scalaApplyEditWait)
	defer timer.Stop()
	select {
	case result := <-completed:
		if result.err != nil {
			return nil, result.err
		}
		if !result.applied {
			return nil, errors.New("metals workspace/applyEdit was declined")
		}
		return response, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for Metals workspace/applyEdit: %w", ctx.Err())
	case <-timer.C:
		return nil, errors.New("metals command completed without a workspace/applyEdit")
	}
}

func (s *scalaProcessSession) endApplyEditCapture() {
	s.mu.Lock()
	s.apply = nil
	s.mu.Unlock()
	s.active.Wait()
}

func (s *scalaProcessSession) handleServerRequest(ctx context.Context, request lsp.Request) (json.RawMessage, *lsp.ErrorObject) {
	if request.Method == "workspace/configuration" {
		return scalaWorkspaceConfiguration(request.Params)
	}
	if request.Method != "workspace/applyEdit" {
		return rejectScalaServerRequest(ctx, request)
	}
	s.mu.Lock()
	apply := s.apply
	if apply != nil {
		s.active.Add(1)
	}
	s.mu.Unlock()
	if apply == nil {
		return scalaApplyEditResponse(false, "no active Scala apply-edit capture")
	}
	defer s.active.Done()
	var params struct {
		Edit json.RawMessage `json:"edit"`
	}
	if err := json.Unmarshal(request.Params, &params); err != nil || len(params.Edit) == 0 || string(params.Edit) == "null" {
		_, _ = apply(json.RawMessage("null"))
		return scalaApplyEditResponse(false, "malformed workspace/applyEdit request")
	}
	applied, err := apply(params.Edit)
	if err != nil {
		return scalaApplyEditResponse(false, err.Error())
	}
	if !applied {
		return scalaApplyEditResponse(false, "captured edit did not satisfy the Scala operation contract")
	}
	return scalaApplyEditResponse(true, "")
}

func scalaApplyEditResponse(applied bool, failure string) (json.RawMessage, *lsp.ErrorObject) {
	response := map[string]any{"applied": applied}
	if failure != "" {
		response["failureReason"] = failure
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return nil, &lsp.ErrorObject{Code: -32603, Message: "encode Scala workspace/applyEdit response"}
	}
	return encoded, nil
}
