package output

import (
	"slices"
	"strings"
	"unicode"

	"github.com/go-git/go-git/v6"
)

const (
	defaultAuthorName  = "memorialiste"
	defaultAuthorEmail = "noreply@local"
	defaultAudience    = "common"
	multiAudienceSlug  = "multi"
)

// branchName formats the auto-generated branch name.
//
// Names are derived entirely from the manifest entry audience values —
// no timestamps, no SHA suffixes. The same audience always produces the
// same branch name, so a re-run while a previous MR is still open will
// fail-fast with ErrBranchExists, prompting the operator to close or
// delete the previous branch before proceeding.
//
//   - All entries share one non-empty audience → "<prefix><slug>"
//   - Entries declare distinct audiences      → "<prefix>multi"
//   - No audience declared on any entry       → "<prefix>common"
func branchName(prefix string, audiences []string) string {
	if prefix == "" {
		prefix = DefaultBranchPrefix
	}
	return prefix + audienceSlug(audiences)
}

// audienceSlug picks the stable slug describing this set of audiences.
func audienceSlug(audiences []string) string {
	var slugs []string
	seen := map[string]struct{}{}
	for _, a := range audiences {
		s := slugify(a)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		slugs = append(slugs, s)
	}
	slices.Sort(slugs)
	switch len(slugs) {
	case 0:
		return defaultAudience
	case 1:
		return slugs[0]
	default:
		return multiAudienceSlug
	}
}

// slugify converts an audience label (e.g. "end users", "AI assistants")
// into a branch-safe slug (lowercase, alphanumeric + dashes, collapsed runs).
func slugify(s string) string {
	var b strings.Builder
	prevDash := true
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
		case !prevDash:
			b.WriteRune('-')
			prevDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// resolveAuthor applies precedence: explicit override → repo config → fallback.
func resolveAuthor(repo *git.Repository, override Author) Author {
	if override.Name != "" || override.Email != "" {
		return override
	}
	if cfg, err := repo.Config(); err == nil && cfg.User.Name != "" {
		return Author{Name: cfg.User.Name, Email: cfg.User.Email}
	}
	return Author{Name: defaultAuthorName, Email: defaultAuthorEmail}
}
