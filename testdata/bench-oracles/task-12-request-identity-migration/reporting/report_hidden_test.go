package reporting

import (
	"example.com/identityservice/request"
	"testing"
)

func TestReportsGroupByRequestIdentity(t *testing.T) {
	rows := Group([]Row{{Request: request.Request{RequestID: "req-hidden"}, Label: "one"}})
	if len(rows["req-hidden"]) != 1 || rows["req-hidden"][0].Label != "one" {
		t.Fatalf("Group() = %#v", rows)
	}
}
