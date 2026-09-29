package review

import "testing"

func TestReviewIDStaysIndependent(t *testing.T) {
	r := Review{ID: "review-hidden", RequestID: "req-hidden"}
	if got := r.Key(); got != "review-hidden" {
		t.Fatalf("Key() = %q, want review's own ID", got)
	}
	if got, ok := ByID([]Review{r}, "review-hidden"); !ok || got.RequestID != "req-hidden" {
		t.Fatalf("ByID() = %#v, %t", got, ok)
	}
	if _, ok := ByID([]Review{r}, "req-hidden"); ok {
		t.Fatal("ByID() matched the correlation RequestID instead of the review ID")
	}
}
