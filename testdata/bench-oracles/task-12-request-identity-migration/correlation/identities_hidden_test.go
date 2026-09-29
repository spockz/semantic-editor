package correlation

import (
	"reflect"
	"testing"
	"example.com/identityservice/request"
	"example.com/identityservice/response"
	"example.com/identityservice/review"
)

func TestTypedLookalikeIDsRemainDistinct(t *testing.T) {
	got := IDs(request.Request{RequestID: "req-1"}, response.Response{ID: "resp-1"}, review.Review{ID: "review-1"})
	want := []string{"req-1", "resp-1", "review-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("IDs() = %#v, want %#v", got, want)
	}
}
