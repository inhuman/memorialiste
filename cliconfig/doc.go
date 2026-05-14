// Package cliconfig defines the CLI flag schema, env-var bindings, and
// cross-field validation rules for the memorialiste command.
//
// The package wraps github.com/alecthomas/kong so that all flag metadata
// (name, env var, default, help, group) lives on the Config struct as
// reflect tags. Parse consumes args and a getenv function and returns a
// populated *Config. Config.Validate runs cross-field rules after parsing.
package cliconfig
