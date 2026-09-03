package service

import (
	"context"
	"testing"
	"time"
)

type fakeBodyLogRepoForCleanup struct {
	GatewayBodyLogRepository

	calls      int
	gotCutoff  time.Time
	deleteErr  error
	deletedCnt int64
}

func (f *fakeBodyLogRepoForCleanup) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	f.calls++
	f.gotCutoff = before
	return f.deletedCnt, f.deleteErr
}

type fakeRetentionProvider struct {
	days int
}

func (f *fakeRetentionProvider) GetGatewayBodyLogRetentionDays(ctx context.Context) int {
	return f.days
}

func TestGatewayBodyLogCleanupServiceDeletesExpired(t *testing.T) {
	repo := &fakeBodyLogRepoForCleanup{deletedCnt: 3}
	svc := NewGatewayBodyLogCleanupService(repo, &fakeRetentionProvider{days: 7})

	before := time.Now().UTC()
	svc.cleanupOnce()

	if repo.calls != 1 {
		t.Fatalf("expected 1 delete call, got %d", repo.calls)
	}
	expectedCutoff := before.AddDate(0, 0, -7)
	if repo.gotCutoff.After(expectedCutoff.Add(2*time.Minute)) || repo.gotCutoff.Before(expectedCutoff.Add(-2*time.Minute)) {
		t.Fatalf("unexpected cutoff: got %v, want ~%v", repo.gotCutoff, expectedCutoff)
	}
}

func TestGatewayBodyLogCleanupServiceSkipsWhenRetentionDisabled(t *testing.T) {
	for _, days := range []int{0, -1} {
		repo := &fakeBodyLogRepoForCleanup{}
		svc := NewGatewayBodyLogCleanupService(repo, &fakeRetentionProvider{days: days})
		svc.cleanupOnce()
		if repo.calls != 0 {
			t.Fatalf("retention=%d: expected no delete call, got %d", days, repo.calls)
		}
	}
}

func TestGatewayBodyLogCleanupServiceNilDepsNeverPanics(t *testing.T) {
	var svc *GatewayBodyLogCleanupService
	svc.Start()
	svc.Stop()
	svc.cleanupOnce()

	svc2 := NewGatewayBodyLogCleanupService(nil, nil)
	svc2.Start()
	svc2.Stop()
	svc2.cleanupOnce()
}

func TestNormalizeGatewayBodyLogRetentionDays(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", GatewayBodyLogRetentionDaysDefault},
		{"7", 7},
		{"30", 30},
		{"0", 0},
		{" 12 ", 12},
		{"abc", GatewayBodyLogRetentionDaysDefault},
		{"-5", GatewayBodyLogRetentionDaysDefault},
		{"9999", GatewayBodyLogRetentionDaysMax},
	}
	for _, c := range cases {
		if got := normalizeGatewayBodyLogRetentionDays(c.raw); got != c.want {
			t.Errorf("normalize(%q) = %d, want %d", c.raw, got, c.want)
		}
	}
}
