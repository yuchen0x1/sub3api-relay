package routes

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/middleware"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterAuthRoutes 注册认证相关路由
func RegisterAuthRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth servermiddleware.JWTAuthMiddleware,
	auditLog servermiddleware.AuditLogMiddleware,
	redisClient *redis.Client,
	settingService *service.SettingService,
	panelRateLimiter *servermiddleware.PanelRateLimiter,
) {
	// 创建速率限制器
	rateLimiter := middleware.NewRateLimiter(redisClient)

	// 公开接口
	auth := v1.Group("/auth")
	auth.Use(servermiddleware.BackendModeAuthGuard(settingService))
	// 认证事件（登录/2FA/token 刷新失败）入审计
	auth.Use(gin.HandlerFunc(auditLog))
	{
		// 登录/2FA 属于高风险入口，增加服务端兜底限流（Redis 故障时 fail-close）
		auth.POST("/login", rateLimiter.LimitWithOptions("auth-login", 20, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.Login)
		auth.POST("/login/2fa", rateLimiter.LimitWithOptions("auth-login-2fa", 20, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.Login2FA)
		auth.POST("/passkey/login/begin", rateLimiter.LimitWithOptions("passkey-login-begin", 20, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Passkey.BeginLogin)
		auth.POST("/passkey/login/finish", rateLimiter.LimitWithOptions("passkey-login-finish", 20, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Passkey.FinishLogin)
		// Token刷新接口添加速率限制：每分钟最多 30 次（Redis 故障时 fail-close）
		auth.POST("/refresh", rateLimiter.LimitWithOptions("refresh-token", 30, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.RefreshToken)
		// 登出接口（公开，允许未认证用户调用以撤销Refresh Token）
		auth.POST("/logout", h.Auth.Logout)
		// 忘记密码接口添加速率限制：每分钟最多 5 次（Redis 故障时 fail-close）
		// 重置密码接口添加速率限制：每分钟最多 10 次（Redis 故障时 fail-close）
		auth.GET("/oauth/linuxdo/start", h.Auth.LinuxDoOAuthStart)
		auth.POST("/oauth/linuxdo/start", rateLimiter.LimitWithOptions("oauth-linuxdo-start", 20, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.LinuxDoOAuthStart)
		auth.GET("/oauth/github/start", h.Auth.GitHubOAuthStart)
		auth.POST("/oauth/github/start", rateLimiter.LimitWithOptions("oauth-github-start", 20, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.GitHubOAuthStart)
		auth.GET("/oauth/github/callback", h.Auth.GitHubOAuthCallback)
		auth.GET("/oauth/google/start", h.Auth.GoogleOAuthStart)
		auth.POST("/oauth/google/start", rateLimiter.LimitWithOptions("oauth-google-start", 20, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.GoogleOAuthStart)
		auth.GET("/oauth/google/callback", h.Auth.GoogleOAuthCallback)
		auth.GET("/oauth/linuxdo/bind/start", func(c *gin.Context) {
			query := c.Request.URL.Query()
			query.Set("intent", "bind_current_user")
			c.Request.URL.RawQuery = query.Encode()
			h.Auth.LinuxDoOAuthStart(c)
		})
		auth.GET("/oauth/linuxdo/callback", h.Auth.LinuxDoOAuthCallback)
		auth.GET("/oauth/wechat/start", h.Auth.WeChatOAuthStart)
		auth.POST("/oauth/wechat/start", rateLimiter.LimitWithOptions("oauth-wechat-start", 20, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.WeChatOAuthStart)
		auth.GET("/oauth/wechat/bind/start", func(c *gin.Context) {
			query := c.Request.URL.Query()
			query.Set("intent", "bind_current_user")
			c.Request.URL.RawQuery = query.Encode()
			h.Auth.WeChatOAuthStart(c)
		})
		auth.GET("/oauth/wechat/callback", h.Auth.WeChatOAuthCallback)
		auth.POST("/oauth/pending/exchange",
			rateLimiter.LimitWithOptions("oauth-pending-exchange", 20, time.Minute, middleware.RateLimitOptions{
				FailureMode: middleware.RateLimitFailClose,
			}),
			h.Auth.ExchangePendingOAuthCompletion,
		)
		auth.POST("/oauth/pending/bind-login",
			rateLimiter.LimitWithOptions("oauth-pending-bind-login", 10, time.Minute, middleware.RateLimitOptions{
				FailureMode: middleware.RateLimitFailClose,
			}),
			h.Auth.BindPendingOAuthLogin,
		)
		auth.POST("/oauth/linuxdo/bind-login",
			rateLimiter.LimitWithOptions("oauth-linuxdo-bind-login", 20, time.Minute, middleware.RateLimitOptions{
				FailureMode: middleware.RateLimitFailClose,
			}),
			h.Auth.BindLinuxDoOAuthLogin,
		)
		auth.POST("/oauth/wechat/bind-login",
			rateLimiter.LimitWithOptions("oauth-wechat-bind-login", 20, time.Minute, middleware.RateLimitOptions{
				FailureMode: middleware.RateLimitFailClose,
			}),
			h.Auth.BindWeChatOAuthLogin,
		)
		auth.GET("/oauth/oidc/start", h.Auth.OIDCOAuthStart)
		auth.POST("/oauth/oidc/start", rateLimiter.LimitWithOptions("oauth-oidc-start", 20, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.OIDCOAuthStart)
		auth.GET("/oauth/oidc/bind/start", func(c *gin.Context) {
			query := c.Request.URL.Query()
			query.Set("intent", "bind_current_user")
			c.Request.URL.RawQuery = query.Encode()
			h.Auth.OIDCOAuthStart(c)
		})
		auth.GET("/oauth/oidc/callback", h.Auth.OIDCOAuthCallback)
		auth.POST("/oauth/oidc/bind-login",
			rateLimiter.LimitWithOptions("oauth-oidc-bind-login", 20, time.Minute, middleware.RateLimitOptions{
				FailureMode: middleware.RateLimitFailClose,
			}),
			h.Auth.BindOIDCOAuthLogin,
		)
		auth.GET("/oauth/dingtalk/start", h.Auth.DingTalkOAuthStart)
		auth.POST("/oauth/dingtalk/start", rateLimiter.LimitWithOptions("oauth-dingtalk-start", 20, time.Minute, middleware.RateLimitOptions{
			FailureMode: middleware.RateLimitFailClose,
		}), h.Auth.DingTalkOAuthStart)
		auth.GET("/oauth/dingtalk/bind/start", func(c *gin.Context) {
			query := c.Request.URL.Query()
			query.Set("intent", "bind_current_user")
			c.Request.URL.RawQuery = query.Encode()
			h.Auth.DingTalkOAuthStart(c)
		})
		auth.GET("/oauth/dingtalk/callback", h.Auth.DingTalkOAuthCallback)
		auth.POST("/oauth/dingtalk/bind-login",
			rateLimiter.LimitWithOptions("oauth-dingtalk-bind-login", 20, time.Minute, middleware.RateLimitOptions{
				FailureMode: middleware.RateLimitFailClose,
			}),
			h.Auth.BindDingTalkOAuthLogin,
		)
	}

	// 公开设置（无需认证）：每次请求都会查询 DB，按客户端 IP 兜底限流，
	// 防止匿名高频刷接口打爆数据库（反代内部地址会被自动跳过，不会误伤）。
	settings := v1.Group("/settings")
	settings.Use(panelRateLimiter.PublicIP())
	{
		settings.GET("/public", h.Setting.GetPublicSettings)
		settings.GET("/email-unsubscribe", h.Setting.UnsubscribeNotificationEmail)
	}

	// 需要认证的当前用户信息
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(servermiddleware.BackendModeUserGuard(settingService))
	// 面板全局按用户限流
	authenticated.Use(panelRateLimiter.Global())
	{
		authenticated.GET("/auth/me", h.Auth.GetCurrentUser)
		// 撤销所有会话（需要认证）
		authenticated.POST("/auth/revoke-all-sessions", h.Auth.RevokeAllSessions)
		authenticated.POST("/auth/oauth/bind-token", h.Auth.PrepareOAuthBindAccessTokenCookie)
	}
}
