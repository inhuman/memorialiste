package openai

import "fmt"

// HTTPError is returned by Complete for non-2xx HTTP responses.
type HTTPError struct {
	// StatusCode is the HTTP status code from the provider.
	StatusCode int
	// Body is a (possibly truncated) excerpt of the response body for debugging.
	Body string
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	return fmt.Sprintf("openai: HTTP %d: %s", e.StatusCode, e.Body)
}
