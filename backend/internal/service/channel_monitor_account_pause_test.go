//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// accountPauseStub 记录 SuspendAccountsByPlatform 调用，模拟账号联动执行器。
type accountPauseStub struct {
	paused []accountPauseCall
}

type accountPauseCall struct {
	platform string
	until    time.Time
	reason   string
}

func (s *accountPauseStub) SuspendAccountsByPlatform(_ context.Context, platform string, until time.Time, reason string) (int64, error) {
	s.paused = append(s.paused, accountPauseCall{platform: platform, until: until, reason: reason})
	return 1, nil
}

func TestMaybeAutoDisable_DoesNotPauseAccountsByDefault(t *testing.T) {
	t.Parallel()

	// AffectsAccounts 默认 false → 即便触发禁用也不联动账号。
	pauser := &accountPauseStub{}
	repo := &autoDisableRepoStub{}
	svc := NewChannelMonitorService(repo, nil)
	svc.SetAutoDisableReader(&autoDisableReaderStub{cfg: OpsChannelMonitorAutoDisableSettings{Enabled: true, ConsecutiveFailures: 1}})
	svc.SetAccountPauser(pauser)

	svc.maybeAutoDisable(context.Background(), &ChannelMonitor{ID: 1, Provider: "openai", Enabled: true}, []*CheckResult{opResult("failed")})
	require.Empty(t, pauser.paused)
}

func TestMaybeAutoDisable_PausesAccountsWhenEnabled(t *testing.T) {
	t.Parallel()

	cfg := OpsChannelMonitorAutoDisableSettings{
		Enabled:             true,
		ConsecutiveFailures: 2,
		AffectsAccounts:     true,
	}
	pauser := &accountPauseStub{}
	svc, repo, _ := newAutoDisableService(cfg)
	svc.SetAccountPauser(pauser)

	monitor := &ChannelMonitor{ID: 9, Name: "gemini", Provider: "gemini", Enabled: true}
	// 第一次失败未达阈值。
	svc.maybeAutoDisable(context.Background(), monitor, []*CheckResult{opResult("failed")})
	require.Empty(t, pauser.paused)

	// 连续第二次失败触发禁用 + 账号联动。
	svc.maybeAutoDisable(context.Background(), monitor, []*CheckResult{opResult("failed")})
	require.Equal(t, []int64{9}, repo.disabledIDs)
	require.Len(t, pauser.paused, 1)
	call := pauser.paused[0]
	require.Equal(t, "gemini", call.platform)
	require.Equal(t, "channel_monitor:9", call.reason)
	// 暂停窗口为 15 分钟，之后自动恢复参与调度。
	require.WithinDuration(t, time.Now().Add(channelMonitorAccountPauseDuration), call.until, 2*time.Second)
}

func TestMaybeAutoDisable_DoesNotPauseWhenProviderEmpty(t *testing.T) {
	t.Parallel()

	cfg := OpsChannelMonitorAutoDisableSettings{
		Enabled:             true,
		ConsecutiveFailures: 1,
		AffectsAccounts:     true,
	}
	pauser := &accountPauseStub{}
	svc, _, _ := newAutoDisableService(cfg)
	svc.SetAccountPauser(pauser)

	// Provider 为空：账号联动跳过，但 monitor 禁用依然生效。
	svc.maybeAutoDisable(context.Background(), &ChannelMonitor{ID: 4, Name: "generic", Enabled: true}, []*CheckResult{opResult("failed")})
	require.Empty(t, pauser.paused)
}
