package service

import (
	"context"
	"errors"
	"sync"
	"time"

	infraerrors "github.com/yuchen0x1/sub3api-relay/internal/pkg/errors"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/logger"
)

var ErrStaticProxyCapacityUnavailable = infraerrors.TooManyRequests("STATIC_PROXY_CAPACITY_UNAVAILABLE", "static proxy capacity unavailable")

// StaticProxyCapacity is the fixed number of concurrent requests allowed by
// each account-owned static proxy assignment.
const StaticProxyCapacity = 10

// Allocation identifies one reserved account/static-proxy request slot.
// ReleaseFunc is idempotent and should be called when the request exits.
type Allocation struct {
	AccountID     int64
	StaticProxyID int64
	RequestID     string
	ReleaseFunc   func()
}

// CalculateStaticProxyAggregateCapacity returns the account-wide capacity for
// the supplied assignment IDs. Invalid and duplicate IDs do not add capacity.
func CalculateStaticProxyAggregateCapacity(staticProxyIDs []int64) int {
	seen := make(map[int64]struct{}, len(staticProxyIDs))
	for _, staticProxyID := range staticProxyIDs {
		if staticProxyID <= 0 {
			continue
		}
		if _, ok := seen[staticProxyID]; ok {
			continue
		}
		seen[staticProxyID] = struct{}{}
	}
	return len(seen) * StaticProxyCapacity
}

func normalizedStaticProxyIDs(staticProxyIDs []int64) []int64 {
	seen := make(map[int64]struct{}, len(staticProxyIDs))
	ids := make([]int64, 0, len(staticProxyIDs))
	for _, staticProxyID := range staticProxyIDs {
		if staticProxyID <= 0 {
			continue
		}
		if _, ok := seen[staticProxyID]; ok {
			continue
		}
		seen[staticProxyID] = struct{}{}
		ids = append(ids, staticProxyID)
	}
	return ids
}

// AcquireStaticProxy reserves one request against both the account aggregate
// and one assigned static proxy. A nil allocation means the account has no
// usable static-proxy assignments. Accounts without static IPs should continue
// through the existing AcquireAccountSlot path unchanged.
func (s *ConcurrencyService) AcquireStaticProxy(ctx context.Context, accountID int64, staticProxyIDs []int64) (*Allocation, error) {
	ids := normalizedStaticProxyIDs(staticProxyIDs)
	if len(ids) == 0 {
		return nil, nil
	}
	if s == nil || s.cache == nil {
		return nil, errors.New("static proxy concurrency cache is unavailable")
	}
	cache, ok := s.cache.(StaticProxyConcurrencyCache)
	if !ok {
		return nil, errors.New("static proxy concurrency cache is unsupported")
	}
	if accountID <= 0 {
		return nil, errors.New("static proxy account id must be greater than zero")
	}

	requestID := generateRequestID()
	maxAggregateConcurrency := len(ids) * StaticProxyCapacity
	for _, staticProxyID := range ids {
		acquired, err := cache.AcquireStaticProxySlot(ctx, accountID, staticProxyID, maxAggregateConcurrency, requestID)
		if err != nil {
			return nil, err
		}
		if acquired {
			return newStaticProxyAllocation(cache, accountID, staticProxyID, requestID), nil
		}
	}
	return nil, nil
}

// AcquireStaticProxyLive reserves a short-lived live request slot using the
// optional live extension. It is kept separate from regular request slots so
// callers that do not use live traffic need no additional cache methods.
func (s *ConcurrencyService) AcquireStaticProxyLive(ctx context.Context, accountID int64, staticProxyIDs []int64) (*Allocation, error) {
	ids := normalizedStaticProxyIDs(staticProxyIDs)
	if len(ids) == 0 {
		return nil, nil
	}
	if s == nil || s.cache == nil {
		return nil, errors.New("static proxy concurrency cache is unavailable")
	}
	cache, ok := s.cache.(StaticProxyLiveConcurrencyCache)
	if !ok {
		return nil, errors.New("static proxy live concurrency cache is unsupported")
	}
	if accountID <= 0 {
		return nil, errors.New("static proxy account id must be greater than zero")
	}

	requestID := generateRequestID()
	maxAggregateConcurrency := len(ids) * StaticProxyCapacity
	for _, staticProxyID := range ids {
		acquired, err := cache.AcquireStaticProxyLiveSlot(ctx, accountID, staticProxyID, maxAggregateConcurrency, requestID)
		if err != nil {
			return nil, err
		}
		if acquired {
			return newStaticProxyLiveAllocation(cache, accountID, staticProxyID, requestID), nil
		}
	}
	return nil, nil
}

func (s *ConcurrencyService) RefreshStaticProxyLive(ctx context.Context, allocation *Allocation) (bool, error) {
	if allocation == nil || allocation.AccountID <= 0 || allocation.StaticProxyID <= 0 || allocation.RequestID == "" {
		return false, nil
	}
	if s == nil || s.cache == nil {
		return false, errors.New("static proxy concurrency cache is unavailable")
	}
	cache, ok := s.cache.(StaticProxyLiveConcurrencyCache)
	if !ok {
		return false, errors.New("static proxy live concurrency cache is unsupported")
	}
	return cache.RefreshStaticProxyLiveSlot(ctx, allocation.AccountID, allocation.StaticProxyID, allocation.RequestID)
}

func (s *ConcurrencyService) GetStaticProxyConcurrency(ctx context.Context, accountID, staticProxyID int64) (int, error) {
	if s == nil || s.cache == nil {
		return 0, nil
	}
	cache, ok := s.cache.(StaticProxyConcurrencyCache)
	if !ok {
		return 0, errors.New("static proxy concurrency cache is unsupported")
	}
	return cache.GetStaticProxyConcurrency(ctx, accountID, staticProxyID)
}

func newStaticProxyAllocation(cache StaticProxyConcurrencyCache, accountID, staticProxyID int64, requestID string) *Allocation {
	var once sync.Once
	return &Allocation{
		AccountID:     accountID,
		StaticProxyID: staticProxyID,
		RequestID:     requestID,
		ReleaseFunc: func() {
			once.Do(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := cache.ReleaseStaticProxySlot(ctx, accountID, staticProxyID, requestID); err != nil {
					logger.LegacyPrintf("service.concurrency", "Warning: failed to release static proxy slot for account %d proxy %d (req=%s): %v", accountID, staticProxyID, requestID, err)
				}
			})
		},
	}
}

func newStaticProxyLiveAllocation(cache StaticProxyLiveConcurrencyCache, accountID, staticProxyID int64, requestID string) *Allocation {
	var once sync.Once
	return &Allocation{
		AccountID:     accountID,
		StaticProxyID: staticProxyID,
		RequestID:     requestID,
		ReleaseFunc: func() {
			once.Do(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := cache.ReleaseStaticProxyLiveSlot(ctx, accountID, staticProxyID, requestID); err != nil {
					logger.LegacyPrintf("service.concurrency", "Warning: failed to release static proxy live slot for account %d proxy %d (req=%s): %v", accountID, staticProxyID, requestID, err)
				}
			})
		},
	}
}
