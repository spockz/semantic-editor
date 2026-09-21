// Package service keeps task-specific delivery acceptance checks outside the model workspace.
package service

import (
	"testing"

	"example.com/auditapp/audit"
)

func TestHiddenRecordNormalizesBeforeDelivery(t *testing.T) {
	sink := audit.NewMemorySink()
	if err := NewDispatcher(sink).Record(audit.Event{ID: "evt-1", Kind: "  PrObE  "}); err != nil {
		t.Fatalf("record event: %v", err)
	}
	if len(sink.Events) != 1 || sink.Events[0].Kind != "probe" {
		t.Fatalf("events = %#v, want normalized probe", sink.Events)
	}
}
