package service

import (
	"context"
	"errors"

	"github.com/yuchen0x1/sub3api-relay/internal/pkg/logger"
)

// applyStaticProxyRouting is the generic gateway request path hook.
//
// For accounts that own enabled static-proxy assignments it:
//  1. Resolves the enabled assignment IDs (populating Account.StaticProxyIDs,
//     the non-persistent runtime-model hint introduced for this feature).
//  2. Reserves one aggregate + per-exit capacity slot through
//     ConcurrencyService.AcquireStaticProxy (see static_proxy_concurrency.go).
//  3. Re-points the account at the selected assignment's existing Proxy so the
//     remainder of GatewayService.Forward forwards through that proxy.
//     Callers MUST release the returned func on every exit path, which frees the
//     reserved slot.
//
// Returns (nil, nil) when static proxy routing does not apply, leaving legacy
// accounts and legacy forwarding completely unchanged.
//
// LIMITATION: only the generic GatewayService.Forward path (the Anthropic-style
// relay used for /v1/messages, Bedrock, and API-key passthrough) applies static
// proxy routing. The OpenAI chat/responses gateways, Gemini-compat, Antigravity,
// Manus, Codex, and WebSocket relays do not consume static-proxy capacity and
// keep using the account's legacy proxy configuration. Extending coverage to
// those protocols is future work; reserve/release semantics already live behind
// ConcurrencyService.AcquireStaticProxy so each protocol can opt in.
func (s *GatewayService) applyStaticProxyRouting(ctx context.Context, account *Account) (func(), error) {
	if s == nil || account == nil || account.ID <= 0 {
		return nil, nil
	}
	if s.staticProxyLookup == nil || s.concurrencyService == nil {
		// No runtime lookup wired or no concurrency backend: legacy path.
		return nil, nil
	}

	targets, err := s.staticProxyLookup.ListEnabledRoutingTargets(ctx, account.ID)
	if err != nil {
		// Fail open to legacy forwarding: a lookup failure must not brick an
		// account that previously routed through its regular proxy.
		logger.LegacyPrintf("service.gateway", "Warning: static proxy routing lookup failed for account %d: %v", account.ID, err)
		return nil, nil
	}
	if len(targets) == 0 {
		// Legacy account without static-proxy assignments.
		return nil, nil
	}

	account.StaticProxyIDs = staticProxyExitIDs(targets)
	allocation, err := s.concurrencyService.AcquireStaticProxy(ctx, account.ID, account.StaticProxyIDs)
	if err != nil {
		// Fail open: if the slot backend cannot be reached, degrade to the
		// legacy path instead of rejecting traffic during an infrastructure
		// outage. Per-exit isolation is advisory in this degraded state.
		logger.LegacyPrintf("service.gateway", "Warning: static proxy slot acquire failed for account %d: %v", account.ID, err)
		return nil, nil
	}
	if allocation == nil {
		// All per-exit slots busy. Fail closed so the account never silently
		// drops back to its legacy/unproxied route while at capacity.
		logger.LegacyPrintf("service.gateway", "Static proxy capacity exhausted for account %d (assignments=%d)", account.ID, len(account.StaticProxyIDs))
		return nil, ErrStaticProxyCapacityUnavailable
	}

	if !s.bindStaticProxyForwardTarget(account, targets, allocation.StaticProxyID) {
		// The reserved exit's proxy disappeared between lookup and reservation;
		// release immediately and fall back to the legacy path.
		allocation.ReleaseFunc()
		return nil, nil
	}
	return allocation.ReleaseFunc, nil
}

func (s *GatewayService) bindStaticProxyForwardTarget(account *Account, targets []StaticProxyRoutingTarget, exitID int64) bool {
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

func staticProxyExitIDs(targets []StaticProxyRoutingTarget) []int64 {
	ids := make([]int64, 0, len(targets))
	for i := range targets {
		ids = append(ids, targets[i].ExitID)
	}
	return ids
}

// staticProxyCapacityUnavailable reports whether err is the capacity-exhaustion
// error produced by the generic gateway path.
func staticProxyCapacityUnavailable(err error) bool {
	return err != nil && errors.Is(err, ErrStaticProxyCapacityUnavailable)
}
