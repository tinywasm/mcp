package mcp

import (
	"context"
)

type PromptCompletionProvider interface {
	// CompletePromptArgument provides completions for a prompt argument
	CompletePromptArgument(ctx context.Context, promptName string, argument CompleteArgument, context CompleteContext) (*Completion, error)
}

type ResourceCompletionProvider interface {
	// CompleteResourceArgument provides completions for a resource template argument
	CompleteResourceArgument(ctx context.Context, uri string, argument CompleteArgument, context CompleteContext) (*Completion, error)
}

// DefaultCompletionProvider returns no completions (fallback)
type DefaultPromptCompletionProvider struct{}

func (p *DefaultPromptCompletionProvider) CompletePromptArgument(ctx context.Context, promptName string, argument CompleteArgument, context CompleteContext) (*Completion, error) {
	return &Completion{
		Values: []string{},
	}, nil
}

// DefaultResourceCompletionProvider returns no completions (fallback)
type DefaultResourceCompletionProvider struct{}

func (p *DefaultResourceCompletionProvider) CompleteResourceArgument(ctx context.Context, uri string, argument CompleteArgument, context CompleteContext) (*Completion, error) {
	return &Completion{
		Values: []string{},
	}, nil
}
