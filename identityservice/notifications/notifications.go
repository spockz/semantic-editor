// Package notifications builds destinations from request identity.
package notifications

import "example.com/identityservice/request"

// Notice is a notification associated with a request.
type Notice struct {
	Request request.Request
}

// DestinationFor returns the notification destination for a request.
func DestinationFor(r request.Request) string {
	return "request:" + r.RequestID
}

// IDs returns the request identities for a batch of notices.
func IDs(notices []Notice) []string {
	ids := make([]string, len(notices))
	for i, n := range notices {
		ids[i] = n.Request.RequestID
	}
	return ids
}
