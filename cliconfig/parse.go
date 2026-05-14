package cliconfig

import (
	"github.com/alecthomas/kong"
)

// getenvResolver resolves flag values from a caller-supplied getenv
// function, looking up the flag's env-tag name(s). Empty values are
// treated as unset so kong falls back to the configured default.
type getenvResolver struct {
	getenv func(string) string
}

func (g *getenvResolver) Validate(_ *kong.Application) error { return nil }

func (g *getenvResolver) Resolve(_ *kong.Context, _ *kong.Path, flag *kong.Flag) (any, error) {
	if g == nil || g.getenv == nil {
		return nil, nil
	}
	for _, name := range flag.Envs {
		if name == "" {
			continue
		}
		if v := g.getenv(name); v != "" {
			return v, nil
		}
	}
	return nil, nil
}

// Parse parses args (typically os.Args[1:]) using kong with env-var
// fallback supplied by getenv. Precedence: defaults < env vars < flags.
//
// When --version is passed, kong prints the version and exits the
// process; Parse never returns in that case.
func Parse(args []string, getenv func(string) string) (*Config, error) {
	return parseWithOptions(args, getenv)
}

func parseWithOptions(args []string, getenv func(string) string, extra ...kong.Option) (*Config, error) {
	var cfg Config
	opts := []kong.Option{
		kong.Name("memorialiste"),
		kong.Description("Documentation update agent — generates and commits doc updates from git diffs via an LLM."),
		kong.Vars{"version": "memorialiste " + Version},
		kong.Resolvers(&getenvResolver{getenv: getenv}),
	}
	opts = append(opts, extra...)

	parser, err := kong.New(&cfg, opts...)
	if err != nil {
		return nil, err
	}
	if _, err := parser.Parse(args); err != nil {
		return nil, err
	}
	return &cfg, nil
}
