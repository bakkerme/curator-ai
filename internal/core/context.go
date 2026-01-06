package core

import "context"

type flowIDKey struct{}
type runIDKey struct{}

func WithFlowID(ctx context.Context, flowID string) context.Context {
	if ctx == nil || flowID == "" {
		return ctx
	}
	return context.WithValue(ctx, flowIDKey{}, flowID)
}

func WithRunID(ctx context.Context, runID string) context.Context {
	if ctx == nil || runID == "" {
		return ctx
	}
	return context.WithValue(ctx, runIDKey{}, runID)
}

func FlowIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(flowIDKey{}).(string); ok {
		return v
	}
	return ""
}

func RunIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(runIDKey{}).(string); ok {
		return v
	}
	return ""
}
