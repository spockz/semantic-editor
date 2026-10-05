// Package review defines the review type with its own identity separate from request identity.
package review

// Review carries a review identity and the correlated request identity.
type Review struct {
	ID        string
	RequestID string
}

// Key returns the review's own identity.
func (r Review) Key() string {
	return r.ID
}

// ByID finds a review by its own ID in the slice.
func ByID(s []Review, id string) (Review, bool) {
	for _, r := range s {
		if r.ID == id {
			return r, true
		}
	}
	return Review{}, false
}
