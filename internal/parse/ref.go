package parse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/may1a/bad-vibes/internal/model"
)

var (
	reNumber = regexp.MustCompile(`^\d+$`)
	reShort  = regexp.MustCompile(`^([\w.\-]+)/([\w.\-]+)#(\d+)$`)
	reURL    = regexp.MustCompile(`^https?://github\.com/([\w.\-]+)/([\w.\-]+)/pull/(\d+)`)
)

// ParseRef parses a PR reference string into a PRRef.
// Supported forms:
//   - "123"                              (bare number; requires defaultRepo "owner/repo")
//   - "owner/repo#123"
//   - "https://github.com/owner/repo/pull/123"
func ParseRef(raw, defaultRepo string) (model.PRRef, error) {
	raw = strings.TrimSpace(raw)

	if reNumber.MatchString(raw) {
		if defaultRepo == "" {
			return model.PRRef{}, fmt.Errorf(
				"PR number %q given but no repo detected — run bv from inside a GitHub repo", raw)
		}
		parts := strings.SplitN(defaultRepo, "/", 2)
		if len(parts) != 2 {
			return model.PRRef{}, fmt.Errorf("invalid default_repo %q (expected owner/repo)", defaultRepo)
		}
		n, _ := strconv.Atoi(raw)
		return model.PRRef{Owner: parts[0], Repo: parts[1], Number: n}, nil
	}

	if m := reShort.FindStringSubmatch(raw); m != nil {
		n, _ := strconv.Atoi(m[3])
		return model.PRRef{Owner: m[1], Repo: m[2], Number: n}, nil
	}

	if m := reURL.FindStringSubmatch(raw); m != nil {
		n, _ := strconv.Atoi(m[3])
		return model.PRRef{Owner: m[1], Repo: m[2], Number: n}, nil
	}

	return model.PRRef{}, fmt.Errorf(
		"unrecognized PR reference %q\n"+
			"  Supported forms: 123  owner/repo#123  https://github.com/owner/repo/pull/123", raw)
}
