package api

import "fmt"

// Exit codes, per the CLI spec.
const (
	ExitOK        = 0
	ExitFailure   = 1 // API-rejected / unclassified, including HTTP 400
	ExitUsage     = 2 // CLI usage / local validation
	ExitAuth      = 3 // credential / token-mint / 401 / 403
	ExitNotFound  = 4 // actual HTTP 404
	ExitRateLimit = 5 // HTTP 429
	ExitUpstream  = 6 // network / TLS / timeout / HTTP 5xx
	ExitInterrupt = 130
)

// CodedError carries the process exit code a failure should map to.
type CodedError struct {
	Code int
	Msg  string
	// Retry indicates a Retry-After value (seconds), surfaced on 429.
	Retry string
	// retryable marks a 401 that a single token refresh may resolve.
	retryable bool
}

func (e *CodedError) Error() string { return e.Msg }

func coded(code int, format string, a ...any) *CodedError {
	return &CodedError{Code: code, Msg: fmt.Sprintf(format, a...)}
}

// ExitCode extracts the exit code from an error, defaulting to ExitFailure.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	if ce, ok := err.(*CodedError); ok {
		return ce.Code
	}
	return ExitFailure
}
