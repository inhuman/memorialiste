// Package fake provides test doubles for memorialiste interfaces.
package fake

import (
	"context"

	"github.com/inhuman/memorialiste/platform"
	"github.com/inhuman/memorialiste/provider"
)

// Provider is a test double for context.Summariser, provider.Provider, and
// platform.Platform.
type Provider struct {
	// SummariseDiffFunc is called by SummariseDiff when non-nil.
	// When nil, SummariseDiff returns the input diff unchanged.
	SummariseDiffFunc func(ctx context.Context, diff string) (string, error)

	// CompleteFunc is called by Complete when non-nil.
	// When nil, Complete returns ("", provider.TokenUsage{}, nil).
	CompleteFunc func(ctx context.Context, messages []provider.Message) (string, provider.TokenUsage, error)

	// PushFunc is called by Push when non-nil.
	// When nil, Push returns nil.
	PushFunc func(ctx context.Context, branch, headSHA string) error

	// OpenChangeRequestFunc is called by OpenChangeRequest when non-nil.
	// When nil, OpenChangeRequest returns a canned result with URL
	// "https://fake/mr/1" and Number 1.
	OpenChangeRequestFunc func(ctx context.Context, req platform.ChangeRequest) (*platform.ChangeResult, error)
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

// Push implements platform.Platform.
func (p *Provider) Push(ctx context.Context, branch, headSHA string) error {
	if p.PushFunc != nil {
		return p.PushFunc(ctx, branch, headSHA)
	}
	return nil
}

// OpenChangeRequest implements platform.Platform.
func (p *Provider) OpenChangeRequest(ctx context.Context, req platform.ChangeRequest) (*platform.ChangeResult, error) {
	if p.OpenChangeRequestFunc != nil {
		return p.OpenChangeRequestFunc(ctx, req)
	}
	return &platform.ChangeResult{URL: "https://fake/mr/1", Number: 1}, nil
}
