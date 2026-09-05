//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBuildAccountForCreateRejectsRemovedPlatform pins the platform removal
// gate: every admin creation funnel (single create, batch, data import,
// duplicate) builds accounts through buildAccountForCreate, so a removed
// platform must be rejected there.
func TestBuildAccountForCreateRejectsRemovedPlatform(t *testing.T) {
	input := &CreateAccountInput{Platform: PlatformAnthropic, Type: AccountTypeAPIKey}
	_, err := buildAccountForCreate(input, nil)
	require.ErrorIs(t, err, ErrAccountPlatformRemoved)

	require.True(t, IsAccountPlatformRemoved(" Anthropic "))
	require.False(t, IsAccountPlatformRemoved(PlatformOpenAI))
	require.False(t, IsAccountPlatformRemoved(PlatformGemini))
	require.False(t, IsAccountPlatformRemoved(""))
}
