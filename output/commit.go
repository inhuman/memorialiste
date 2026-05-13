package output

import (
	"time"

	"github.com/go-git/go-git/v6"
)

const (
	timestampFormat    = "20060102-150405"
	defaultAuthorName  = "memorialiste"
	defaultAuthorEmail = "noreply@local"
)

// branchName formats the auto-generated branch name as
// "<prefix>YYYYMMDD-HHMMSS-<7chars>" using UTC time.
func branchName(prefix string, now time.Time, headHash string) string {
	if prefix == "" {
		prefix = DefaultBranchPrefix
	}
	short := headHash
	if len(short) > 7 {
		short = short[:7]
	}
	return prefix + now.UTC().Format(timestampFormat) + "-" + short
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
