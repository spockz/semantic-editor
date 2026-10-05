// Package scheduler filters jobs by request identity.
package scheduler

import "example.com/identityservice/request"

// Job is a scheduled job associated with a request.
type Job struct {
	Request request.Request
}

// Pending returns jobs matching the request identity.
func Pending(jobs []Job, r request.Request) []Job {
	var result []Job
	for _, j := range jobs {
		if j.Request.RequestID == r.RequestID {
			result = append(result, j)
		}
	}
	return result
}
