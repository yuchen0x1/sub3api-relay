package service

import "context"

// UpdateChannel identifies the source used for project updates.
type UpdateChannel string

const (
	UpdateChannelGitHub UpdateChannel = "github"
	// GitHubUpdateRepo is the canonical repository for project releases and updates.
	GitHubUpdateRepo = "yuchen0x1/sub3api-relay"
)

// ResolveUpdateChannel always returns the canonical GitHub update channel.
func ResolveUpdateChannel(ctx context.Context) (UpdateChannel, error) {
	_ = ctx
	return UpdateChannelGitHub, nil
}
