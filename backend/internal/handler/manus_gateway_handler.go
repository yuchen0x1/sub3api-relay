package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuchen0x1/sub3api-relay/internal/server/middleware"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

type ManusGatewayHandler struct {
	gatewayService    *service.GatewayService
	manusService      *service.ManusGatewayService
	concurrencyHelper *ConcurrencyHelper
}

func NewManusGatewayHandler(
	gatewayService *service.GatewayService,
	manusService *service.ManusGatewayService,
	concurrencyService *service.ConcurrencyService,
) *ManusGatewayHandler {
	return &ManusGatewayHandler{
		gatewayService:    gatewayService,
		manusService:      manusService,
		concurrencyHelper: NewConcurrencyHelper(concurrencyService, SSEPingFormatNone, 0),
	}
}

func (h *ManusGatewayHandler) Proxy(c *gin.Context) {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		writeManusError(c, http.StatusUnauthorized, "permission_denied", "Invalid API key")
		return
	}
	endpoint := strings.TrimPrefix(c.Param("endpoint"), "/")
	selection, err := h.gatewayService.SelectAccountWithLoadAwareness(
		c.Request.Context(), apiKey.GroupID, "", "", nil, "", 0,
	)
	if err != nil {
		markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
		writeManusError(c, http.StatusServiceUnavailable, "no_available_account", "No available Manus accounts")
		return
	}
	account := selection.Account
	setOpsSelectedAccount(c, account.ID, account.Platform)

	release := selection.ReleaseFunc
	if !selection.Acquired {
		if selection.WaitPlan == nil {
			writeManusError(c, http.StatusServiceUnavailable, "no_available_account", "No available Manus accounts")
			return
		}
		streamStarted := false
		release, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(
			c, account.ID, selection.WaitPlan.MaxConcurrency, selection.WaitPlan.Timeout, false, &streamStarted,
		)
		if err != nil {
			writeManusError(c, http.StatusTooManyRequests, "rate_limited", err.Error())
			return
		}
	}
	if release != nil {
		defer release()
	}

	resp, err := h.manusService.Forward(
		c.Request.Context(), account, c.Request.Method, endpoint, c.Request.URL.RawQuery, c.Request.Header, c.Request.Body,
	)
	if err != nil {
		writeManusError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if err := h.manusService.WriteResponse(c.Writer, resp); err != nil {
		_ = c.Error(err)
	}
}

func writeManusError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"ok": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
