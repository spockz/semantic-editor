// Package audit keeps task-specific acceptance checks outside the model workspace.
package audit

import "testing"

func TestHiddenAuditDeliveryRules(t *testing.T) {
	if got := NormalizeKind("  PrObE  "); got != "probe" {
		t.Fatalf("normalized kind = %q, want probe", got)
	}
	if got := Classify("probe"); got != "control" {
		t.Fatalf("classification = %q, want control", got)
	}
}
