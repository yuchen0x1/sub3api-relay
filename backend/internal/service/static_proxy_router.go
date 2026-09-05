package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/logger"
)

// acquireStaticProxyRouting is the protocol-agnostic core of static-proxy exit
// routing. It reserves one aggregate + per-exit capacity slot for the account
// and re-points account.Proxy at the selected exit's proxy so every downstream
// transport build (generic, OpenAI/Codex, Gemini-compat, Antigravity, Manus,
// WebSocket) forwards through that exit.
//
// Failure semantics:
//   - lookup / slot-backend errors fail OPEN to the legacy proxy path so an
//     infrastructure outage never bricks an account;
//   - capacity exhaustion fails CLOSED with ErrStaticProxyCapacityUnavailable
//     so the account never silently drops to an unthrottled route.
func acquireStaticProxyRouting(ctx context.Context, lookup StaticProxyRoutingLookup, concurrency *ConcurrencyService, account *Account) (func(), error) {
	if lookup == nil || concurrency == nil || account == nil || account.ID <= 0 {
		return nil, nil
	}

	targets, err := lookup.ListEnabledRoutingTargets(ctx, account.ID)
	if err != nil {
		logger.LegacyPrintf("service.gateway", "Warning: static proxy routing lookup failed for account %d: %v", account.ID, err)
		return nil, nil
	}
	if len(targets) == 0 {
		// Legacy account without static-proxy assignments.
		return nil, nil
	}

	account.StaticProxyIDs = staticProxyExitIDs(targets)
	allocation, err := concurrency.AcquireStaticProxy(ctx, account.ID, account.StaticProxyIDs)
	if err != nil {
		logger.LegacyPrintf("service.gateway", "Warning: static proxy slot acquire failed for account %d: %v", account.ID, err)
		return nil, nil
	}
	if allocation == nil {
		// All per-exit slots busy. Fail closed so the account never silently
		// drops back to its legacy/unproxied route while at capacity.
		logger.LegacyPrintf("service.gateway", "Static proxy capacity exhausted for account %d (assignments=%d)", account.ID, len(account.StaticProxyIDs))
		return nil, ErrStaticProxyCapacityUnavailable
	}

	if !bindStaticProxyForwardTarget(account, targets, allocation.StaticProxyID) {
		// The reserved exit's proxy disappeared between lookup and reservation;
		// release immediately and fall back to the legacy path.
		allocation.ReleaseFunc()
		return nil, nil
	}
	return allocation.ReleaseFunc, nil
}

func bindStaticProxyForwardTarget(account *Account, targets []StaticProxyRoutingTarget, exitID int64) bool {
	if account == nil {
		return false
	}
	for i := range targets {
		target := &targets[i]
		if target.ExitID != exitID {
			continue
		}
		if target.Proxy == nil || target.ProxyID <= 0 {
			return false
		}
		account.Proxy = target.Proxy
		proxyID := target.ProxyID
		account.ProxyID = &proxyID
		return true
	}
	return false
}

// applyStaticProxyRoutingForProtocol applies routing on behalf of a protocol
// gateway service that owns a gin context (Antigravity, Gemini-compat). On
// capacity exhaustion it writes the same Claude-style 429 body the generic
// forward path writes, then returns the sentinel error. Callers MUST invoke
// the returned release func on every exit path.
func applyStaticProxyRoutingForProtocol(ctx context.Context, router *StaticProxyRouter, c *gin.Context, account *Account) (func(), error) {
	if router == nil || account == nil || account.ID <= 0 {
		return nil, nil
	}
	release, err := router.Apply(ctx, account)
	if err != nil {
		if errors.Is(err, ErrStaticProxyCapacityUnavailable) && c != nil {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"type": "error",
				"error": gin.H{
					"type":    "rate_limit_error",
					"message": "Static proxy capacity exceeded, please retry later",
				},
			})
		}
		return nil, err
	}
	return release, nil
}

// StaticProxyRouter applies static-proxy exit routing for one gateway request.
// A single shared instance is wired into every gateway service (generic,
// OpenAI/Codex, Gemini-compat, Antigravity, Manus, WebSocket) so per-exit
// capacity — 10 concurrency slots per enabled exit — is enforced on every
// relay path, not just the generic forward.
type StaticProxyRouter struct {
	lookup         StaticProxyRoutingLookup
	concurrency    *ConcurrencyService
	settingService *SettingService
}

func NewStaticProxyRouter(
	lookup StaticProxyRoutingLookup,
	concurrency *ConcurrencyService,
	settingService *SettingService,
) *StaticProxyRouter {
	return &StaticProxyRouter{
		lookup:         lookup,
		concurrency:    concurrency,
		settingService: settingService,
	}
}

// Enabled reports whether the static_ip_pool canary flag allows routing.
// Without a settings service the flag defaults to enabled: accounts that
// configure exits keep their v0.1.198 behavior, and the flag exists as an
// emergency rollback switch, not as an opt-in gate.
func (r *StaticProxyRouter) Enabled(ctx context.Context) bool {
	if r == nil || r.settingService == nil {
		return true
	}
	return r.settingService.GetStaticIPPoolEnabled(ctx)
}

// Apply reserves one aggregate + per-exit slot for the request and binds the
// account to the selected exit's proxy. Returns (nil, nil) when static routing
// does not apply (flag off, no wiring, or a legacy account), leaving the
// caller's forwarding path completely unchanged. Callers MUST invoke the
// returned release func on every exit path.
func (r *StaticProxyRouter) Apply(ctx context.Context, account *Account) (func(), error) {
	if r == nil || account == nil || account.ID <= 0 {
		return nil, nil
	}
	if r.lookup == nil || r.concurrency == nil {
		return nil, nil
	}
	if !r.Enabled(ctx) {
		return nil, nil
	}
	return acquireStaticProxyRouting(ctx, r.lookup, r.concurrency, account)
}
