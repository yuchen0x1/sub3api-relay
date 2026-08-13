package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type identityUACacheStub struct {
	fp      *Fingerprint
	deleted []int64
}

func (s *identityUACacheStub) GetFingerprint(_ context.Context, _ int64) (*Fingerprint, error) {
	if s.fp == nil {
		return nil, nil
	}
	cp := *s.fp
	return &cp, nil
}

func (s *identityUACacheStub) DeleteFingerprint(_ context.Context, accountID int64) error {
	s.deleted = append(s.deleted, accountID)
	return nil
}

func (s *identityUACacheStub) SetFingerprint(_ context.Context, _ int64, fp *Fingerprint) error {
	if fp == nil {
		s.fp = nil
		return nil
	}
	cp := *fp
	s.fp = &cp
	return nil
}

func (s *identityUACacheStub) GetMaskedSessionID(_ context.Context, _ int64) (string, error) {
	return "", nil
}

func (s *identityUACacheStub) SetMaskedSessionID(_ context.Context, _ int64, _ string) error {
	return nil
}

func TestIsAcceptableFingerprintUserAgent(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want bool
	}{
		{"official default", "claude-cli/2.1.220 (external, cli)", true},
		{"bare version", "claude-cli/2.1.220", true},
		{"non claude product form valid", "some-agent/1.2.3", true},
		{"empty", "", false},
		{"local suffix", "claude-cli/2.1.220-local", false},
		{"dev suffix", "claude-cli/2.1.220-dev", false},
		{"sentinel major", "claude-cli/999.0.0", false},
		{"browser ua", "Mozilla/5.0", false},
		{"too long", strings.Repeat("a", 300) + "/1.2.3", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isAcceptableFingerprintUserAgent(tt.ua))
		})
	}
}

func TestGetOrCreateFingerprint_RejectsMalformedUAOnCreate(t *testing.T) {
	cache := &identityUACacheStub{}
	svc := NewIdentityService(cache)

	headers := http.Header{}
	headers.Set("User-Agent", "claude-cli/999.0.0-local")

	fp, err := svc.GetOrCreateFingerprint(context.Background(), 1, headers)
	require.NoError(t, err)
	require.NotNil(t, fp)
	require.Equal(t, defaultFingerprint.UserAgent, fp.UserAgent)
	require.NotEmpty(t, fp.ClientID)
}

func TestGetOrCreateFingerprint_HealsPoisonedCache(t *testing.T) {
	cache := &identityUACacheStub{fp: &Fingerprint{
		ClientID:  "existing-client",
		UserAgent: "claude-cli/999.0.0-local",
	}}
	svc := NewIdentityService(cache)

	// 无有效 UA 时，直接回退默认指纹，避免畸形 UA 继续毒化上游。
	headers := http.Header{}
	fp, err := svc.GetOrCreateFingerprint(context.Background(), 1, headers)
	require.NoError(t, err)
	require.NotNil(t, fp)
	require.Equal(t, defaultFingerprint.UserAgent, fp.UserAgent)
	require.Equal(t, "existing-client", fp.ClientID)
	require.Equal(t, defaultFingerprint.UserAgent, cache.fp.UserAgent)
}

func TestAccountUsageService_ResetFingerprint(t *testing.T) {
	cache := &identityUACacheStub{fp: &Fingerprint{ClientID: "client-1", UserAgent: defaultFingerprint.UserAgent}}
	svc := &AccountUsageService{identityCache: cache}

	err := svc.ResetFingerprint(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, []int64{42}, cache.deleted)
}

func TestAccountUsageService_ResetFingerprint_NoCache(t *testing.T) {
	svc := &AccountUsageService{}
	err := svc.ResetFingerprint(context.Background(), 42)
	require.Error(t, err)
}
