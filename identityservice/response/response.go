// Package response defines the response type with its own identity separate from request identity.
package response

// Response carries a response identity and the correlated request identity.
type Response struct {
	ID        string
	RequestID string
}

// CorrelationID returns the response's own identity.
func (r Response) CorrelationID() string {
	return r.ID
}

// Index finds a response by its own ID in the slice.
func Index(s []Response, id string) (Response, bool) {
	for _, r := range s {
		if r.ID == id {
			return r, true
		}
	}
	return Response{}, false
}

// AuditRow carries an independent identity for audit purposes.
type AuditRow struct {
	ID string
}

// SameID returns true when two independent identities match.
func SameID(r Response, a AuditRow) bool {
	return r.ID == a.ID
}
