//go:build unit

package service

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type updateServiceCacheStub struct {
	data string
}

func (s *updateServiceCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if s.data == "" {
		return "", errors.New("cache miss")
	}
	return s.data, nil
}

func (s *updateServiceCacheStub) SetUpdateInfo(_ context.Context, data string, _ time.Duration) error {
	s.data = data
	return nil
}

type updateServiceGitHubClientStub struct {
	release        *GitHubRelease
	recentReleases []*GitHubRelease
	recentErr      error
}

func (s *updateServiceGitHubClientStub) FetchLatestRelease(context.Context, string) (*GitHubRelease, error) {
	return s.release, nil
}

func (s *updateServiceGitHubClientStub) FetchRecentReleases(context.Context, string, int) ([]*GitHubRelease, error) {
	return s.recentReleases, s.recentErr
}

func (s *updateServiceGitHubClientStub) DownloadFile(context.Context, string, string, int64) error {
	panic("DownloadFile should not be called when no update is available")
}

func (s *updateServiceGitHubClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	panic("FetchChecksumFile should not be called when no update is available")
}

func TestUpdateServicePerformUpdateNoUpdateReturnsSentinel(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{
			release: &GitHubRelease{
				TagName: "v0.1.132",
				Name:    "v0.1.132",
			},
		},
		"0.1.132",
		"release",
	)

	err := svc.PerformUpdate(context.Background())

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoUpdateAvailable))
	require.ErrorIs(t, err, ErrNoUpdateAvailable)
}

func TestResolveUpdateChannelUsesGitHub(t *testing.T) {
	channel, err := ResolveUpdateChannel(context.Background())
	require.NoError(t, err)
	require.Equal(t, UpdateChannelGitHub, channel)
	require.Equal(t, "yuchen0x1/sub3api-relay", GitHubUpdateRepo)
}

func TestUpdateServiceOlderVersionsStillReceiveUpdateNotifications(t *testing.T) {
	for _, version := range []string{"0.4.4", "0.5.0", "0.5.6"} {
		for _, buildType := range []string{"release", "source", "docker"} {
			t.Run(version+"/"+buildType, func(t *testing.T) {
				svc := NewUpdateService(
					&updateServiceCacheStub{},
					&updateServiceGitHubClientStub{release: &GitHubRelease{TagName: "v0.5.7"}},
					version,
					buildType,
				)

				info, err := svc.CheckUpdate(context.Background(), false)

				require.NoError(t, err)
				require.True(t, info.HasUpdate)
				require.Equal(t, version, info.CurrentVersion)
				require.Equal(t, "0.5.7", info.LatestVersion)
				require.Equal(t, buildType, info.BuildType)
			})
		}
	}
}

func TestUpdateServiceRejectsInPlaceUpdateOutsideStandaloneRelease(t *testing.T) {
	for _, buildType := range []string{"source", "docker"} {
		t.Run(buildType, func(t *testing.T) {
			svc := NewUpdateService(
				&updateServiceCacheStub{},
				&updateServiceGitHubClientStub{},
				"0.1.180",
				buildType,
			)

			require.ErrorIs(t, svc.PerformUpdate(context.Background()), ErrInPlaceUpdateUnsupported)
			require.ErrorIs(t, svc.Rollback(), ErrInPlaceUpdateUnsupported)
			require.ErrorIs(t, svc.RollbackToVersion(context.Background(), "0.1.179"), ErrInPlaceUpdateUnsupported)
		})
	}
}

func TestUpdateServiceInPlaceUpdatePlatformSupport(t *testing.T) {
	tests := []struct {
		name      string
		buildType string
		goos      string
		wantErr   error
	}{
		{name: "release linux", buildType: "release", goos: "linux"},
		{name: "release windows", buildType: "release", goos: "windows", wantErr: ErrWindowsInPlaceUpdateUnsupported},
		{name: "docker windows", buildType: "docker", goos: "windows", wantErr: ErrInPlaceUpdateUnsupported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ensureInPlaceUpdateSupported(tt.buildType, tt.goos)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func newRollbackTestService(current string, releases []*GitHubRelease) *UpdateService {
	return NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{recentReleases: releases},
		current,
		"release",
	)
}

func TestUpdateServiceListRollbackVersionsFiltersAndCaps(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.148", PublishedAt: "2026-07-09T00:00:00Z"},                       // newer than current: excluded
		{TagName: "v0.1.147", PublishedAt: "2026-07-08T00:00:00Z"},                       // current: excluded
		{TagName: "v0.1.146-rc1", PublishedAt: "2026-07-07T12:00:00Z", Prerelease: true}, // prerelease: excluded
		{TagName: "v0.1.146'$(touch /tmp/nope)'", PublishedAt: "2026-07-07T06:00:00Z"},   // invalid: excluded
		{TagName: "v0.1.146", PublishedAt: "2026-07-07T00:00:00Z"},
		{TagName: "v0.1.145", PublishedAt: "2026-07-06T00:00:00Z", Draft: true}, // draft: excluded
		{TagName: "v0.1.144", PublishedAt: "2026-07-05T00:00:00Z"},
		{TagName: "v0.1.144", PublishedAt: "2026-07-05T00:00:00Z"}, // duplicate: excluded
		{TagName: "v0.1.143", PublishedAt: "2026-07-04T00:00:00Z"},
		{TagName: "v0.1.142", PublishedAt: "2026-07-03T00:00:00Z"}, // beyond cap of 3: excluded
	}
	svc := newRollbackTestService("0.1.147", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Len(t, versions, 3)
	require.Equal(t, "0.1.146", versions[0].Version)
	require.Equal(t, "0.1.144", versions[1].Version)
	require.Equal(t, "0.1.143", versions[2].Version)
}

func TestUpdateServiceRollbackCannotCrossCredentialDigestMigration(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.5.8"},
		{TagName: "v0.5.7"},
		{TagName: "v0.5.6"},
		{TagName: "v0.5.5"},
	}
	svc := newRollbackTestService("0.5.9", releases)

	versions, err := svc.ListRollbackVersions(context.Background())
	require.NoError(t, err)
	require.Equal(t, []RollbackVersion{{Version: "0.5.8"}, {Version: "0.5.7"}}, versions)
	require.ErrorIs(t, svc.RollbackToVersion(context.Background(), "v0.5.6"), ErrRollbackRequiresDatabaseRestore)
	if runtime.GOOS != "windows" {
		require.ErrorIs(t, svc.Rollback(), ErrRollbackRequiresDatabaseRestore)
	}

	for _, test := range []struct {
		current, target string
		blocked         bool
	}{
		{"0.5.6", "0.5.5", false},
		{"0.5.7", "0.5.6", true},
		{"0.5.8", "0.5.7", false},
		{"0.6.0", "0.4.4", true},
	} {
		require.Equal(t, test.blocked, rollbackRequiresDatabaseRestore(test.current, test.target))
	}
}

func TestUpdateServiceRejectsInvalidLatestReleaseVersion(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{release: &GitHubRelease{TagName: "v0.1.182'$(id)'"}},
		"0.1.181",
		"release",
	)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.Equal(t, "0.1.181", info.LatestVersion)
	require.False(t, info.HasUpdate)
	require.Contains(t, info.Warning, "invalid release version")
}

func TestUpdateServiceRejectsNilLatestRelease(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{},
		"0.1.181",
		"release",
	)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.False(t, info.HasUpdate)
	require.Equal(t, "0.1.181", info.LatestVersion)
	require.Contains(t, info.Warning, "empty latest release")
}

func TestUpdateServiceArchiveAssetMatchingRejectsSidecars(t *testing.T) {
	require.True(t, isReleaseArchiveAsset("sub3api_0.5.6_windows_amd64.tar.gz", "windows_amd64"))
	for _, name := range []string{
		"sub3api_0.5.6_windows_amd64.tar.gz.sig",
		"sub3api_0.5.6_windows_amd64.tar.gz.sha256",
		"nested/sub3api_0.5.6_windows_amd64.tar.gz",
		"sub3api_0.5.6_windows_amd64.txt",
	} {
		require.False(t, isReleaseArchiveAsset(name, "windows_amd64"), name)
	}
}

func TestUpdateServicePerformUpdateRejectsReleaseWithoutAssets(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{release: &GitHubRelease{TagName: "v0.5.7"}},
		"0.5.6",
		"release",
	)

	err := svc.PerformUpdate(context.Background())
	if runtime.GOOS == "windows" {
		require.ErrorIs(t, err, ErrWindowsInPlaceUpdateUnsupported)
		return
	}
	require.ErrorContains(t, err, "latest release has no downloadable assets")
}

func TestUpdateServiceRequiresChecksumAsset(t *testing.T) {
	svc := NewUpdateService(nil, nil, "0.5.6", "release")
	assetName := "sub3api_0.5.7_" + svc.getArchiveName() + ".tar.gz"
	err := svc.applyReleaseAssets(context.Background(), []Asset{{
		Name:        assetName,
		DownloadURL: "https://github.com/yuchen0x1/sub3api-relay/releases/download/v0.5.7/" + assetName,
	}})
	require.ErrorContains(t, err, "release is missing checksums.txt")
}

func TestValidateDownloadURLRejectsLookalikeHostsAndPorts(t *testing.T) {
	require.NoError(t, validateDownloadURL("https://github.com/yuchen0x1/sub3api-relay/releases/download/v0.5.6/checksums.txt"))
	require.NoError(t, validateDownloadURL("https://objects.githubusercontent.com/github-production-release-asset/checksums.txt"))
	for _, rawURL := range []string{
		"https://github.com.evil.example/release.tar.gz",
		"https://objects.githubusercontent.com.evil.example/release.tar.gz",
		"https://github.com:8443/release.tar.gz",
		"https://user@github.com/release.tar.gz",
	} {
		require.Error(t, validateDownloadURL(rawURL), rawURL)
	}
}

func TestStrictReleaseVersion(t *testing.T) {
	for _, version := range []string{"0.1.182", "10.20.30"} {
		require.True(t, isStrictReleaseVersion(version), version)
	}
	for _, version := range []string{"", "v0.1.182", "0.1", "0.1.182-rc1", "0.1.182'$(id)'", "０.１.１８２", "0.1.99999999999999999999999999"} {
		require.False(t, isStrictReleaseVersion(version), version)
	}
}

func TestUpdateServiceRejectsOverflowingLatestReleaseVersion(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{release: &GitHubRelease{TagName: "v0.1.99999999999999999999999999"}},
		"0.1.181",
		"release",
	)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.False(t, info.HasUpdate)
	require.Equal(t, "0.1.181", info.LatestVersion)
	require.Contains(t, info.Warning, "invalid release version")
}

func TestUpdateServiceListRollbackVersionsSortsUnorderedInput(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.144"},
		{TagName: "v0.1.146"},
		{TagName: "v0.1.145"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Len(t, versions, 3)
	require.Equal(t, "0.1.146", versions[0].Version)
	require.Equal(t, "0.1.145", versions[1].Version)
	require.Equal(t, "0.1.144", versions[2].Version)
}

func TestUpdateServiceListRollbackVersionsEmptyWhenNoneOlder(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.147"},
		{TagName: "v0.1.148"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Empty(t, versions)
}

func TestUpdateServiceListRollbackVersionsPropagatesFetchError(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{recentErr: errors.New("github unavailable")},
		"0.1.147",
		"release",
	)

	_, err := svc.ListRollbackVersions(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "github unavailable")
}

func TestUpdateServiceRollbackToVersionRejectsDisallowedTargets(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.148"},
		{TagName: "v0.1.147"},
		{TagName: "v0.1.146"},
		{TagName: "v0.1.145"},
		{TagName: "v0.1.144"},
		{TagName: "v0.1.143"},
		{TagName: "v0.1.142"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	for _, target := range []string{
		"",         // empty
		"0.1.147",  // current version
		"v0.1.147", // current version with prefix
		"0.1.148",  // newer than current
		"0.1.142",  // older than the 3 most recent
		"9.9.9",    // nonexistent
	} {
		err := svc.RollbackToVersion(context.Background(), target)
		require.ErrorIs(t, err, ErrRollbackVersionNotAllowed, "target %q should be rejected", target)
	}
}

func TestUpdateServiceRollbackToVersionAcceptsVPrefix(t *testing.T) {
	// No platform asset in the release: the target passes the allowlist check
	// and fails later at asset lookup, proving the version itself was accepted.
	releases := []*GitHubRelease{
		{TagName: "v0.1.147"},
		{TagName: "v0.1.146"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	err := svc.RollbackToVersion(context.Background(), "v0.1.146")
	if runtime.GOOS == "windows" {
		require.ErrorIs(t, err, ErrWindowsInPlaceUpdateUnsupported)
		return
	}

	require.Error(t, err)
	require.NotErrorIs(t, err, ErrRollbackVersionNotAllowed)
	require.Contains(t, err.Error(), "no compatible release found")
}
