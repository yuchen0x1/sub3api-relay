import type { SystemSettings, DefaultPlatformQuotasMap } from "@/api/admin/settings";

// SettingsView 及其 Tab 子组件共用的表单结构：SystemSettings 去掉由
// wechat 能力推导覆盖的三字段，并补充各 secret 的本地明文输入槽。
export type SettingsForm = Omit<
  SystemSettings,
  | "wechat_connect_open_enabled"
  | "wechat_connect_mp_enabled"
  | "wechat_connect_mobile_enabled"
  | "tencent_captcha_enabled"
  | "tencent_captcha_app_id"
  | "tencent_captcha_app_secret_key_configured"
  | "tencent_captcha_cloud_secret_id_configured"
  | "tencent_captcha_cloud_secret_key_configured"
  | "tencent_captcha_region"
  | "aliyun_captcha_enabled"
  | "aliyun_captcha_access_key_id"
  | "aliyun_captcha_access_key_secret_configured"
  | "aliyun_captcha_scene_id"
  | "aliyun_captcha_prefix"
  | "aliyun_captcha_region"
  | "openai_codex_client_version"
  | "openai_codex_client_version_synced"
  | "openai_codex_version_auto_sync_enabled"
> & {
  smtp_password: string;
  turnstile_secret_key: string;
  linuxdo_connect_client_secret: string;
  dingtalk_connect_client_secret: string;
  wechat_connect_app_secret: string;
  wechat_connect_open_app_secret: string;
  wechat_connect_mp_app_secret: string;
  wechat_connect_mobile_app_secret: string;
  wechat_connect_open_enabled: boolean;
  wechat_connect_mp_enabled: boolean;
  wechat_connect_mobile_enabled: boolean;
  oidc_connect_client_secret: string;
  github_oauth_client_secret: string;
  google_oauth_client_secret: string;
  force_email_on_third_party_signup: boolean;
  openai_low_upstream_rate_priority_enabled: boolean;
  openai_oauth_scheduling_rate_multiplier: number;
  openai_advanced_scheduler_enabled: boolean;
  openai_advanced_scheduler_sticky_weighted_enabled: boolean;
  openai_advanced_scheduler_subscription_priority_enabled: boolean;
  openai_advanced_scheduler_lb_top_k: string;
  openai_advanced_scheduler_weight_priority: string;
  openai_advanced_scheduler_weight_load: string;
  openai_advanced_scheduler_weight_queue: string;
  openai_advanced_scheduler_weight_error_rate: string;
  openai_advanced_scheduler_weight_ttft: string;
  openai_advanced_scheduler_weight_reset: string;
  openai_advanced_scheduler_weight_quota_headroom: string;
  openai_advanced_scheduler_weight_upstream_cost: string;
  openai_advanced_scheduler_weight_previous_response: string;
  openai_advanced_scheduler_weight_session_sticky: string;
  // 系统全局平台限额 map；form 内始终归一化为全 4 平台对象（模板非空绑定依赖此不变量）
  default_platform_quotas: DefaultPlatformQuotasMap;
  tencent_captcha_enabled?: boolean;
  tencent_captcha_app_id?: string;
  tencent_captcha_app_secret_key_configured?: boolean;
  tencent_captcha_cloud_secret_id_configured?: boolean;
  tencent_captcha_cloud_secret_key_configured?: boolean;
  tencent_captcha_region?: string;
  aliyun_captcha_enabled?: boolean;
  aliyun_captcha_access_key_id?: string;
  aliyun_captcha_access_key_secret_configured?: boolean;
  aliyun_captcha_scene_id?: string;
  aliyun_captcha_prefix?: string;
  aliyun_captcha_region?: string;
  openai_codex_client_version?: string;
  openai_codex_client_version_synced?: string;
  openai_codex_version_auto_sync_enabled?: boolean;
};
