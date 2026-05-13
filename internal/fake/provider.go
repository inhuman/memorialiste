// Package fake provides test doubles for memorialiste interfaces.
package fake

import (
	"context"

	"github.com/inhuman/memorialiste/provider"
)

// Provider is a test double for context.Summariser AND provider.Provider.
type Provider struct {
	// SummariseDiffFunc is called by SummariseDiff when non-nil.
	// When nil, SummariseDiff returns the input diff unchanged.
	SummariseDiffFunc func(ctx context.Context, diff string) (string, error)

	// CompleteFunc is called by Complete when non-nil.
	// When nil, Complete returns ("", provider.TokenUsage{}, nil).
	CompleteFunc func(ctx context.Context, messages []provider.Message) (string, provider.TokenUsage, error)
}

// SummariseDiff implements context.Summariser.
func (p *Provider) SummariseDiff(ctx context.Context, diff string) (string, error) {
	if p.SummariseDiffFunc != nil {
		return p.SummariseDiffFunc(ctx, diff)
	}
	return diff, nil
}

// Complete implements provider.Provider.
func (p *Provider) Complete(ctx context.Context, messages []provider.Message) (string, provider.TokenUsage, error) {
	if p.CompleteFunc != nil {
		return p.CompleteFunc(ctx, messages)
	}
	return "", provider.TokenUsage{}, nil
}
