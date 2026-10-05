// Package correlation collects identities from distinct types.
package correlation

import (
	"example.com/identityservice/request"
	"example.com/identityservice/response"
	"example.com/identityservice/review"
)

// IDs returns the identity strings from a request, response, and review.
func IDs(r request.Request, resp response.Response, rev review.Review) []string {
	return []string{r.RequestID, resp.ID, rev.ID}
}
