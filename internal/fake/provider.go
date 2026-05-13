// Package fake provides test doubles for memorialiste interfaces.
package fake

import "context"

// Provider is a test double for context.Summariser.
type Provider struct {
	// SummariseDiffFunc is called by SummariseDiff when non-nil.
	// When nil, SummariseDiff returns the input diff unchanged.
	SummariseDiffFunc func(ctx context.Context, diff string) (string, error)
}

// SummariseDiff implements context.Summariser.
func (p *Provider) SummariseDiff(ctx context.Context, diff string) (string, error) {
	if p.SummariseDiffFunc != nil {
		return p.SummariseDiffFunc(ctx, diff)
	}
	return diff, nil
}
