package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoginAbuseProtectorLocksIPAfterRepeatedFailures(t *testing.T) {
	now := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	protector := NewLoginAbuseProtector()
	protector.now = func() time.Time { return now }

	for i := 1; i < defaultLoginIPFailureThreshold; i++ {
		_, locked := protector.RecordLoginFailure("13.158.34.225", "admin@sub2api.local")
		require.False(t, locked, "attempt %d should not lock yet", i)
	}

	lock, locked := protector.RecordLoginFailure("13.158.34.225", "admin@sub2api.local")
	require.True(t, locked)
	require.Equal(t, "ip", lock.Scope)
	require.Equal(t, defaultLoginLockDuration, lock.RetryAfter)

	lock, locked = protector.CheckLoginLocked("13.158.34.225", "admin@sub2api.local")
	require.True(t, locked)
	require.Equal(t, "ip", lock.Scope)
	require.Equal(t, defaultLoginLockDuration, lock.RetryAfter)
}

func TestLoginAbuseProtectorLocksIdentifierAcrossIPs(t *testing.T) {
	now := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	protector := NewLoginAbuseProtector()
	protector.now = func() time.Time { return now }

	for i := 1; i <= defaultLoginIDFailureThreshold; i++ {
		_, _ = protector.RecordLoginFailure("198.51.100."+string(rune('0'+i%10)), "Admin@Sub2API.Local ")
	}

	lock, locked := protector.CheckLoginLocked("203.0.113.9", "admin@sub2api.local")
	require.True(t, locked)
	require.Equal(t, "identifier", lock.Scope)
	require.Equal(t, defaultLoginLockDuration, lock.RetryAfter)
}

func TestLoginAbuseProtectorResetOnSuccess(t *testing.T) {
	now := time.Date(2026, 8, 13, 15, 0, 0, 0, time.UTC)
	protector := NewLoginAbuseProtector()
	protector.now = func() time.Time { return now }

	for i := 1; i < defaultLoginIPFailureThreshold; i++ {
		_, _ = protector.RecordLoginFailure("13.158.34.225", "admin@sub2api.local")
	}
	protector.ResetLoginFailures("13.158.34.225", "admin@sub2api.local")

	_, locked := protector.RecordLoginFailure("13.158.34.225", "admin@sub2api.local")
	require.False(t, locked)
}

func TestLoginTemporarilyLockedErrorIncludesRetryMetadata(t *testing.T) {
	err := LoginTemporarilyLockedError(LoginLockout{Scope: "ip", RetryAfter: 90 * time.Second})

	require.Equal(t, int32(429), err.Code)
	require.Equal(t, "AUTH_LOGIN_TEMPORARILY_LOCKED", err.Reason)
	require.Equal(t, "90", err.Metadata["retry_after_seconds"])
	require.Equal(t, "ip", err.Metadata["scope"])
}
