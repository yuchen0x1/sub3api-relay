//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type autoDisableRepoStub struct {
	ChannelMonitorRepository
	disabledIDs []int64
}

func (r *autoDisableRepoStub) SetEnabled(_ context.Context, id int64, enabled bool) error {
	if !enabled {
		r.disabledIDs = append(r.disabledIDs, id)
	}
	return nil
}

type autoDisableSchedulerStub struct {
	unscheduled []int64
}

func (s *autoDisableSchedulerStub) Schedule(m *ChannelMonitor) {}

func (s *autoDisableSchedulerStub) Unschedule(id int64) {
	s.unscheduled = append(s.unscheduled, id)
}

type autoDisableReaderStub struct {
	cfg OpsChannelMonitorAutoDisableSettings
}

func (r *autoDisableReaderStub) ChannelMonitorAutoDisableSettings() OpsChannelMonitorAutoDisableSettings {
	return r.cfg
}

func newAutoDisableService(cfg OpsChannelMonitorAutoDisableSettings) (*ChannelMonitorService, *autoDisableRepoStub, *autoDisableSchedulerStub) {
	repo := &autoDisableRepoStub{}
	sched := &autoDisableSchedulerStub{}
	svc := NewChannelMonitorService(repo, nil)
	svc.SetAutoDisableReader(&autoDisableReaderStub{cfg: cfg})
	svc.SetScheduler(sched)
	return svc, repo, sched
}

func opResult(status string) *CheckResult {
	return &CheckResult{Status: status}
}

func TestMonitorRoundFailed(t *testing.T) {
	t.Parallel()

	require.False(t, monitorRoundFailed(nil))
	require.False(t, monitorRoundFailed([]*CheckResult{opResult("operational")}))
	require.False(t, monitorRoundFailed([]*CheckResult{opResult("degraded")}))
	require.True(t, monitorRoundFailed([]*CheckResult{opResult("failed")}))
	require.True(t, monitorRoundFailed([]*CheckResult{opResult("error")}))
	// 任一模型可用即不算整体失败。
	require.False(t, monitorRoundFailed([]*CheckResult{opResult("failed"), opResult("operational")}))
}

func TestMaybeAutoDisable_DisabledByDefault(t *testing.T) {
	t.Parallel()

	// 未设置 reader（等价默认关闭）→ 不触发禁用。
	repo := &autoDisableRepoStub{}
	svc := NewChannelMonitorService(repo, nil)
	svc.maybeAutoDisable(context.Background(), &ChannelMonitor{ID: 1, Enabled: true}, []*CheckResult{opResult("failed")})
	require.Empty(t, repo.disabledIDs)

	// 配置里 Enabled=false → 不触发。
	svc2, repo2, _ := newAutoDisableService(OpsChannelMonitorAutoDisableSettings{Enabled: false, ConsecutiveFailures: 3})
	svc2.maybeAutoDisable(context.Background(), &ChannelMonitor{ID: 2, Enabled: true}, []*CheckResult{opResult("failed")})
	require.Empty(t, repo2.disabledIDs)
}

func TestMaybeAutoDisable_DisablesAfterThreshold(t *testing.T) {
	t.Parallel()

	svc, repo, sched := newAutoDisableService(OpsChannelMonitorAutoDisableSettings{Enabled: true, ConsecutiveFailures: 2})
	monitor := &ChannelMonitor{ID: 7, Name: "openai", Enabled: true}

	// 第一次失败：未达阈值。
	svc.maybeAutoDisable(context.Background(), monitor, []*CheckResult{opResult("failed")})
	require.Empty(t, repo.disabledIDs)
	require.Empty(t, sched.unscheduled)

	// 成功一轮：计数器复位。
	svc.maybeAutoDisable(context.Background(), monitor, []*CheckResult{opResult("operational")})

	// 连续两次失败：触发禁用 + 取消调度。
	svc.maybeAutoDisable(context.Background(), monitor, []*CheckResult{opResult("error")})
	svc.maybeAutoDisable(context.Background(), monitor, []*CheckResult{opResult("error")})
	require.Equal(t, []int64{7}, repo.disabledIDs)
	require.Equal(t, []int64{7}, sched.unscheduled)
}

func TestMaybeAutoDisable_DoesNotDisableAlreadyDisabled(t *testing.T) {
	t.Parallel()

	svc, repo, sched := newAutoDisableService(OpsChannelMonitorAutoDisableSettings{Enabled: true, ConsecutiveFailures: 1})
	svc.maybeAutoDisable(context.Background(), &ChannelMonitor{ID: 3, Enabled: false}, []*CheckResult{opResult("failed")})
	require.Empty(t, repo.disabledIDs)
	require.Empty(t, sched.unscheduled)
}

func TestChannelMonitorAutoDisableDefaultConfig(t *testing.T) {
	validAdvanced := func() *OpsAdvancedSettings {
		return &OpsAdvancedSettings{AutoRefreshIntervalSec: 30}
	}
	t.Parallel()

	cfg := defaultOpsAdvancedSettings()
	require.False(t, cfg.ChannelMonitorAutoDisable.Enabled)
	require.Equal(t, 3, cfg.ChannelMonitorAutoDisable.ConsecutiveFailures)

	// normalize 会把 <1 的阈值回填为 3。
	cfg2 := &OpsAdvancedSettings{ChannelMonitorAutoDisable: OpsChannelMonitorAutoDisableSettings{Enabled: true, ConsecutiveFailures: 0}}
	normalizeOpsAdvancedSettings(cfg2)
	require.Equal(t, 3, cfg2.ChannelMonitorAutoDisable.ConsecutiveFailures)

	// 启用但阈值为 0 → 校验失败。
	err := validateOpsAdvancedSettings(validAdvanced())
	err = validateOpsAdvancedSettings(&OpsAdvancedSettings{ChannelMonitorAutoDisable: OpsChannelMonitorAutoDisableSettings{Enabled: true, ConsecutiveFailures: 0}, AutoRefreshIntervalSec: 30})
	require.Error(t, err)

	// 合法配置通过。
	err = validateOpsAdvancedSettings(&OpsAdvancedSettings{ChannelMonitorAutoDisable: OpsChannelMonitorAutoDisableSettings{Enabled: true, ConsecutiveFailures: 5}, AutoRefreshIntervalSec: 30})
	require.NoError(t, err)
}
