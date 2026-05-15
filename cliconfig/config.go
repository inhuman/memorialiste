package cliconfig

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
)

// Config holds the full runtime configuration. All fields are populated
// by Parse from defaults < env vars < command-line flags.
type Config struct {
	ProviderURL  string `name:"provider-url"  env:"MEMORIALISTE_PROVIDER_URL"  default:"http://localhost:11434" help:"OpenAI-compatible base URL" group:"Provider"`
	Model        string `name:"model"         env:"MEMORIALISTE_MODEL"         default:"qwen3-coder:30b"        help:"Model tag" group:"Provider"`
	ModelParams  string `name:"model-params"  env:"MEMORIALISTE_MODEL_PARAMS"  default:""                       help:"Extra model parameters as JSON object" group:"Provider"`
	APIKey       string `name:"api-key"       env:"MEMORIALISTE_API_KEY"       default:""                       help:"Bearer token for the LLM provider (optional)" group:"Provider"`
	SystemPrompt string `name:"system-prompt" env:"MEMORIALISTE_SYSTEM_PROMPT" default:""                       help:"System prompt: literal string, @path/to/file, or empty for built-in" group:"Provider"`
	Prompt       string `name:"prompt"        env:"MEMORIALISTE_PROMPT"        default:""                       help:"Additional user prompt appended after diff context" group:"Provider"`
	Language     string `name:"language"      env:"MEMORIALISTE_LANGUAGE"      default:"english"                help:"Output language" group:"Provider"`

	DocStructure string `name:"doc-structure" env:"MEMORIALISTE_DOC_STRUCTURE" default:"docs/.docstructure.yaml" help:"Path to the doc structure manifest" group:"Doc Structure"`
	RepoPath     string `name:"repo"          env:"MEMORIALISTE_REPO"          default:"."                       help:"Path to the local git repository root" group:"Doc Structure"`

	TokenBudget  int    `name:"token-budget"  env:"MEMORIALISTE_TOKEN_BUDGET"  default:"12000"              help:"Max tokens for diff context before summarisation" group:"Output"`
	DryRun       bool   `name:"dry-run"       env:"MEMORIALISTE_DRY_RUN"       default:"true"               help:"Write files locally; skip branch+commit and platform calls" group:"Output"`
	BranchPrefix string `name:"branch-prefix" env:"MEMORIALISTE_BRANCH_PREFIX" default:"docs/memorialiste-" help:"Prefix for the auto-generated branch name" group:"Output"`
	ASTContext   bool   `name:"ast-context"   env:"MEMORIALISTE_AST_CONTEXT"   default:"false"              help:"Enable AST-enriched diff context via grep-ast" group:"Output"`
	RepoMeta     string `name:"repo-meta"     env:"MEMORIALISTE_REPO_META"     default:"basic"              help:"Repository metadata level in LLM context: basic (default) or extended" enum:"basic,extended" group:"Output"`

	CodeSearch         bool `name:"code-search"           env:"MEMORIALISTE_CODE_SEARCH"           default:"false" help:"Enable AST code-search tool for the LLM" group:"Tools"`
	CodeSearchMaxTurns int  `name:"code-search-max-turns" env:"MEMORIALISTE_CODE_SEARCH_MAX_TURNS" default:"10"    help:"Max tool-call turns before aborting" group:"Tools"`

	Platform      string `name:"platform"       env:"MEMORIALISTE_PLATFORM"       default:"gitlab" help:"VCS platform: gitlab or github" group:"Platform"`
	PlatformURL   string `name:"platform-url"   env:"MEMORIALISTE_PLATFORM_URL"   default:""       help:"Platform base URL (for self-hosted instances)" group:"Platform"`
	PlatformToken string `name:"platform-token" env:"MEMORIALISTE_PLATFORM_TOKEN" default:""       help:"Personal access token" group:"Platform"`
	ProjectID     string `name:"project-id"     env:"MEMORIALISTE_PROJECT_ID"     default:""       help:"GitLab project ID or GitHub owner/repo" group:"Platform"`
	BaseBranch    string `name:"base-branch"    env:"MEMORIALISTE_BASE_BRANCH"    default:"main"   help:"Target base branch for MR/PR" group:"Platform"`

	Version kong.VersionFlag `name:"version" help:"Show version and exit"`
}

// ValidationError aggregates all validation failures into a single error.
type ValidationError struct {
	Messages []string
}

// Error returns all messages joined with newlines, each prefixed "error: ".
func (v *ValidationError) Error() string {
	lines := make([]string, len(v.Messages))
	for i, m := range v.Messages {
		lines[i] = "error: " + m
	}
	return strings.Join(lines, "\n")
}

// Validate runs cross-field validation rules. Returns *ValidationError
// aggregating all violations; nil when valid.
func (c *Config) Validate() error {
	var msgs []string

	switch c.Platform {
	case "gitlab", "github":
	default:
		msgs = append(msgs, fmt.Sprintf("--platform must be one of: gitlab, github (got %q)", c.Platform))
	}

	if c.TokenBudget <= 0 {
		msgs = append(msgs, fmt.Sprintf("--token-budget must be > 0 (got %d)", c.TokenBudget))
	}

	if !c.DryRun {
		if c.PlatformToken == "" {
			msgs = append(msgs, "--platform-token (or MEMORIALISTE_PLATFORM_TOKEN) is required when --dry-run=false")
		}
		if c.ProjectID == "" {
			msgs = append(msgs, "--project-id (or MEMORIALISTE_PROJECT_ID) is required when --dry-run=false")
		}
	}

	if strings.HasPrefix(c.SystemPrompt, "@") {
		path := strings.TrimPrefix(c.SystemPrompt, "@")
		if _, err := os.Stat(path); err != nil {
			msgs = append(msgs, fmt.Sprintf("--system-prompt file %q: %v", path, err))
		}
	}

	if len(msgs) == 0 {
		return nil
	}
	return &ValidationError{Messages: msgs}
}
