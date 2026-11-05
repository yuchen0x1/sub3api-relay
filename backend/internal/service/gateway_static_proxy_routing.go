package service

import (
	"context"
	"errors"
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
// Coverage: the same SharedProxyRouter semantics are applied by every gateway
// protocol path (OpenAI chat/responses/Codex/Grok, Gemini-compat, Antigravity,
// Manus, and the OpenAI WebSocket relays) through the shared
// StaticProxyRouter — each protocol service calls StaticProxyRouter.Apply at
// its Forward entry and releases on every exit path. The static_ip_pool
// setting gates all paths uniformly.
func (s *GatewayService) applyStaticProxyRouting(ctx context.Context, account *Account) (func(), error) {
	if s == nil || account == nil || account.ID <= 0 {
		return nil, nil
	}
	if s.staticProxyLookup == nil || s.concurrencyService == nil {
		// No runtime lookup wired or no concurrency backend: legacy path.
		return nil, nil
	}
	if s.settingService != nil && !s.settingService.GetStaticIPPoolEnabled(ctx) {
		return nil, nil
	}
	return acquireStaticProxyRouting(ctx, s.staticProxyLookup, s.concurrencyService, account)
}

//nolint:unused // 由 gateway_static_proxy_routing_test.go 覆盖的装配助手
func (s *GatewayService) bindStaticProxyForwardTarget(account *Account, targets []StaticProxyRoutingTarget, exitID int64) bool {
	return bindStaticProxyForwardTarget(account, targets, exitID)
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
//nolint:unused // 由 gateway_static_proxy_routing_test.go 覆盖的错误分类助手
func staticProxyCapacityUnavailable(err error) bool {
	return err != nil && errors.Is(err, ErrStaticProxyCapacityUnavailable)
}
