package response

import "testing"

func TestResponseAndRequestIDsStayDistinct(t *testing.T) {
	r := Response{ID: "resp-hidden", RequestID: "req-hidden"}
	if got := r.CorrelationID(); got != "resp-hidden" {
		t.Fatalf("CorrelationID() = %q, want response's own ID", got)
	}
	if got, ok := Index([]Response{r}, "resp-hidden"); !ok || got.RequestID != "req-hidden" {
		t.Fatalf("Index() = %#v, %t", got, ok)
	}
	if _, ok := Index([]Response{r}, "req-hidden"); ok {
		t.Fatal("Index() matched the correlation RequestID instead of the response ID")
	}
}

func TestSameIDUsesIndependentIDs(t *testing.T) {
	if SameID(Response{ID: "resp-hidden", RequestID: "req-hidden"}, AuditRow{ID: "req-hidden"}) {
		t.Fatal("SameID() matched Response.RequestID instead of Response.ID")
	}
	if !SameID(Response{ID: "shared", RequestID: "req-hidden"}, AuditRow{ID: "shared"}) {
		t.Fatal("SameID() did not match equal independent IDs")
	}
}
