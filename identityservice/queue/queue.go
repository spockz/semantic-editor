// Package queue provides queue operations keyed by request identity.
package queue

import "example.com/identityservice/request"

// Entry is a queue entry keyed by a string and carrying a request.
type Entry struct {
	Key   string
	Value request.Request
}

// Contains reports whether the queue contains an entry matching the request identity.
func Contains(entries []Entry, r request.Request) bool {
	for _, e := range entries {
		if e.Key == r.RequestID {
			return true
		}
	}
	return false
}

// BatchKeys returns the request identities for a batch of requests.
func BatchKeys(reqs []request.Request) []string {
	keys := make([]string, len(reqs))
	for i, r := range reqs {
		keys[i] = r.RequestID
	}
	return keys
}
