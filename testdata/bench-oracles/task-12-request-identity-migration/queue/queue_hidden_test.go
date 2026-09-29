package queue

import (
	"example.com/identityservice/request"
	"reflect"
	"testing"
)

func TestQueueUsesRequestIdentity(t *testing.T) {
	r := request.Request{RequestID: "req-hidden"}
	if !Contains([]Entry{{Key: "req-hidden", Value: request.Request{RequestID: "different-request"}}}, r) {
		t.Fatal("Contains() did not use the entry key and request identity")
	}
	if got := BatchKeys([]request.Request{r}); !reflect.DeepEqual(got, []string{"req-hidden"}) {
		t.Fatalf("BatchKeys() = %#v", got)
	}
}
