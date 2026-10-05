// Package reporting groups rows by request identity.
package reporting

import "example.com/identityservice/request"

// Row is a reporting row associated with a request.
type Row struct {
	Request request.Request
	Label   string
}

// Group groups rows by the request identity.
func Group(rows []Row) map[string][]Row {
	groups := make(map[string][]Row)
	for _, r := range rows {
		groups[r.Request.RequestID] = append(groups[r.Request.RequestID], r)
	}
	return groups
}
