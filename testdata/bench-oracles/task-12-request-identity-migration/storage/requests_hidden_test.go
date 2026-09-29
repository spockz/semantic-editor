package storage

import (
	"example.com/identityservice/request"
	"testing"
)

func TestLookupUsesRecordIDAndRenamedRequestField(t *testing.T) {
	want := RequestRecord{ID: "record-1", Request: request.Request{RequestID: "embedded-req"}}
	got, ok := Lookup([]RequestRecord{want}, request.Request{RequestID: "record-1"})
	if !ok || got.ID != "record-1" || got.Request.RequestID != "embedded-req" {
		t.Fatalf("Lookup() = %#v, %t", got, ok)
	}
	if _, ok := Lookup([]RequestRecord{want}, request.Request{RequestID: "embedded-req"}); ok {
		t.Fatal("Lookup() matched the embedded request identity instead of the record ID")
	}
}
