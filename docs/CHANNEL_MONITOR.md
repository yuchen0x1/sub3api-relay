# Channel Monitor

渠道监控（Channel Monitor）对配置的上游端点按间隔执行健康检测，为运维提供渠道可用性、延迟与状态视图。

## 自动禁用（Auto-disable）

一条监控在 `OpsSettings` 中开启自动禁用并配置连续失败阈值后：

- 连续 N 轮检测整体失败 → 自动把该监控置为 `disabled` 并取消其调度任务。
- 检测到任一轮成功 → 连续失败计数复位。
- 进程内计数，进程重启会清零。

### 联动暂停同平台账号调度（Affects accounts）

若同时勾选「联动暂停同平台账号调度」（`channel_monitor_auto_disable.affects_accounts`，默认关闭）：

- 连续失败触发自动禁用时，会按监控的 `Provider` 把所有 active 账号临时标记为不可调度（写入 `temp_unschedulable_until = now + 15min`，reason 为 `channel_monitor:<monitor_id>`）。
- 15 分钟后过期，账号自动恢复参与路由，无需人工干预。
- 账号列表状态列会展示「渠道监控暂停，HH:mm:ss 恢复」来源提示。
- 过期后自动恢复，不写 `schedulable` 持久状态，也不改变账号 `status`。

> 与冷却（cooldown）的区别：冷却只是临时跳过 N 分钟的检测调度，不影响网关路由。
> 自动禁用影响监控自身的启停；账号联动额外影响同平台账号的路由资格，适合“上游端点整体不可用”的长期故障告警。
