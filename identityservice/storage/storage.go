// Package storage provides storage operations keyed by record identity.
package storage

import "example.com/identityservice/request"

// RequestRecord is a storage record with its own identity and an embedded request.
type RequestRecord struct {
	ID      string
	Request request.Request
}

// Lookup finds a record by its own ID in the slice.
func Lookup(records []RequestRecord, req request.Request) (RequestRecord, bool) {
	for _, r := range records {
		if r.ID == req.RequestID {
			return r, true
		}
	}
	return RequestRecord{}, false
}
