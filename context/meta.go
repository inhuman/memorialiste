package context

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
)

// MetaLevel controls which fields RepoMeta carries and which appear in the
// formatted block.
type MetaLevel string

// Supported metadata levels.
const (
	MetaBasic    MetaLevel = "basic"
	MetaExtended MetaLevel = "extended"
)

// TagInfo is one tag entry in RecentTags.
type TagInfo struct {
	// Name is the short ref name, e.g. "v0.2.0".
	Name string
	// Date is the committer timestamp of the tagged commit.
	Date time.Time
}

// RepoMeta is the metadata bundle for one generation run.
type RepoMeta struct {
	// LatestTag is the most recent tag by commit date; empty when no tags exist.
	LatestTag string
	// HeadSHA is the full 40-char hex commit SHA at HEAD.
	HeadSHA string
	// ShortSHA is the first 7 chars of HeadSHA.
	ShortSHA string
	// RemoteURL is the redacted URL of the `origin` remote; populated only for MetaExtended.
	RemoteURL string
	// Branch is the current branch name, or "(detached)"; populated only for MetaExtended.
	Branch string
	// RecentTags carries up to 5 tags in reverse-chronological order; populated only for MetaExtended.
	RecentTags []TagInfo
}

// Format produces the text block to prepend to the LLM user message.
// The output is deterministic for a given (RepoMeta, level) pair.
// Returns "" when m is nil.
func (m *RepoMeta) Format(level MetaLevel) string {
	if m == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString("=== Repository metadata ===\n")
	if m.LatestTag == "" {
		b.WriteString("Latest tag: (none)\n")
	} else {
		b.WriteString("Latest tag: " + m.LatestTag + "\n")
	}
	b.WriteString("HEAD: " + m.HeadSHA + "\n")
	b.WriteString("Short SHA: " + m.ShortSHA + "\n")

	if level == MetaExtended {
		remote := m.RemoteURL
		if remote == "" {
			remote = "(none)"
		} else {
			remote = redactURL(remote)
		}
		b.WriteString("Remote: " + remote + "\n")

		branch := m.Branch
		if branch == "" {
			branch = "(detached)"
		}
		b.WriteString("Branch: " + branch + "\n")

		if len(m.RecentTags) > 0 {
			b.WriteString("Recent tags:\n")
			for _, t := range m.RecentTags {
				fmt.Fprintf(&b, "- %s (%s)\n", t.Name, t.Date.Format("2006-01-02"))
			}
		}
	}

	b.WriteString("=== End metadata ===")
	return b.String()
}

// redactURL replaces the password component of a URL with "<redacted>"
// (which will be percent-encoded). Non-parseable strings and URLs without
// user info (e.g. SSH git@host:owner/repo.git) pass through unchanged.
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	u.User = url.UserPassword(u.User.Username(), "<redacted>")
	return u.String()
}

// collectTags lists every tag in the repository, resolves annotated tags
// to their target commit, and returns the result sorted by commit date
// descending. Dangling refs are silently skipped.
func collectTags(repo *git.Repository) ([]TagInfo, error) {
	iter, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("context: list tags: %w", err)
	}

	var tags []TagInfo
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		hash := ref.Hash()
		if tagObj, tagErr := repo.TagObject(hash); tagErr == nil {
			hash = tagObj.Target
		}
		commit, commitErr := repo.CommitObject(hash)
		if commitErr != nil {
			return nil
		}
		tags = append(tags, TagInfo{
			Name: ref.Name().Short(),
			Date: commit.Committer.When,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("context: iterate tags: %w", err)
	}

	slices.SortFunc(tags, func(a, b TagInfo) int {
		return b.Date.Compare(a.Date)
	})
	return tags, nil
}

// gatherRepoMeta collects HEAD + tag metadata, and (for MetaExtended) the
// origin remote URL, branch name, and recent tags. Individual sub-step
// failures (e.g. no origin remote) leave the corresponding field empty.
func gatherRepoMeta(repo *git.Repository, level MetaLevel) (*RepoMeta, error) {
	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("context: resolve HEAD: %w", err)
	}
	headSHA := headRef.Hash().String()
	short := headSHA
	if len(short) > 7 {
		short = short[:7]
	}

	tags, err := collectTags(repo)
	if err != nil {
		return nil, err
	}

	m := &RepoMeta{
		HeadSHA:  headSHA,
		ShortSHA: short,
	}
	if len(tags) > 0 {
		m.LatestTag = tags[0].Name
	}

	if level == MetaExtended {
		if remote, remErr := repo.Remote("origin"); remErr == nil {
			if cfg := remote.Config(); cfg != nil && len(cfg.URLs) > 0 {
				m.RemoteURL = redactURL(cfg.URLs[0])
			}
		}
		if headRef.Name().IsBranch() {
			m.Branch = headRef.Name().Short()
		} else {
			m.Branch = "(detached)"
		}
		n := len(tags)
		if n > 5 {
			n = 5
		}
		if n > 0 {
			m.RecentTags = append([]TagInfo(nil), tags[:n]...)
		}
	}

	return m, nil
}
