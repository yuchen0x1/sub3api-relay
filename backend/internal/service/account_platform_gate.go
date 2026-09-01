package service

import (
	"errors"
	"strings"
)

// ErrAccountPlatformRemoved is returned when an admin operation tries to put
// an account onto a platform that has been removed from the product surface.
var ErrAccountPlatformRemoved = errors.New("platform removed: anthropic accounts are no longer supported")

// removedAccountPlatforms enumerates account platforms removed from the
// product surface. Every account-creation funnel (create, batch, import,
// duplicate, OAuth completion) must reject them so a removed platform can
// never re-enter the system through the admin API. Protocol-level code that
// still references the platform (compat endpoints, pricing maps, quota rows)
// is intentionally kept — see the platform removal notes in DEV_GUIDE.md.
var removedAccountPlatforms = map[string]struct{}{
	PlatformAnthropic: {},
}

// IsAccountPlatformRemoved reports whether the given account platform has
// been removed from the product surface.
func IsAccountPlatformRemoved(platform string) bool {
	_, ok := removedAccountPlatforms[strings.ToLower(strings.TrimSpace(platform))]
	return ok
}
