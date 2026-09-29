package notifications

import (
	"reflect"
	"testing"
	"example.com/identityservice/request"
)

func TestNotificationRequestIdentity(t *testing.T) {
	r := request.Request{RequestID: "req-hidden"}
	if got := DestinationFor(r); got != "request:req-hidden" {
		t.Fatalf("DestinationFor() = %q", got)
	}
	if got := IDs([]Notice{{Request: r}}); !reflect.DeepEqual(got, []string{"req-hidden"}) {
		t.Fatalf("IDs() = %#v", got)
	}
}
