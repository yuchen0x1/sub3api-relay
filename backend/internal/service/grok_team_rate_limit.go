package service

import (
	"context"
	"strings"
)

// grokTeamRateLimitModelContextKey carries the upstream model for rate-limit
// side effects (team+model cool). Safe when model is empty.
type grokTeamRateLimitModelContextKey struct{}

// withGrokTeamRateLimitModel attaches the upstream model name for rate-limit
// side effects (team+model cool). Safe when model is empty.
func withGrokTeamRateLimitModel(ctx context.Context, model string) context.Context {
	model = strings.TrimSpace(model)
	if model == "" || ctx == nil {
		return ctx
	}
	return context.WithValue(ctx, grokTeamRateLimitModelContextKey{}, model)
}

func grokRequestedModelFromCtx(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	model, _ := ctx.Value(grokTeamRateLimitModelContextKey{}).(string)
	return strings.TrimSpace(model)
}
