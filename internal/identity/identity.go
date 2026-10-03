package identity

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
)

// Assumption: whatever sits in front of this service has already authenticated
// the caller and sets this header from the verified certificate identity
const Header = "X-Workload-Id"

var (
	ErrMissing = errors.New("missing workload identity")
	ErrInvalid = errors.New("invalid workload identity")
	idPattern  = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

func FromRequest(r *http.Request) (string, error) {
	workload := strings.TrimSpace(r.Header.Get(Header))
	if workload == "" {
		return "", ErrMissing
	}
	// Restrict to safe charset; without restriction attacker could inject new lines, control chars etc.
	if !idPattern.MatchString(workload) {
		return "", ErrInvalid
	}
	return workload, nil
}
