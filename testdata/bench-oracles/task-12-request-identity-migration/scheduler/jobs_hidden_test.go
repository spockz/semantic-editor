package scheduler

import (
	"testing"
	"example.com/identityservice/request"
)

func TestPendingMatchesRequestIdentity(t *testing.T) {
	want := Job{Request: request.Request{RequestID: "req-hidden"}}
	got := Pending([]Job{want, {Request: request.Request{RequestID: "other"}}}, request.Request{RequestID: "req-hidden"})
	if len(got) != 1 || got[0].Request.RequestID != "req-hidden" {
		t.Fatalf("Pending() = %#v", got)
	}
}
