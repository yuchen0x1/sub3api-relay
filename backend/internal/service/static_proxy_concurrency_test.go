//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type staticProxyConcurrencyCacheFake struct {
	ConcurrencyCache

	acquire func(context.Context, int64, int64, int, string) (bool, error)
	release func(context.Context, int64, int64, string) error
	count   func(context.Context, int64, int64) (int, error)
	refresh func(context.Context, int64, int64, string) (bool, error)

	acquireCalls []staticProxyAcquireCall
	releaseCalls []staticProxyReleaseCall
}

type staticProxyAcquireCall struct {
	accountID     int64
	staticProxyID int64
	capacity      int
	requestID     string
}

type staticProxyReleaseCall struct {
	accountID     int64
	staticProxyID int64
	requestID     string
}

func (f *staticProxyConcurrencyCacheFake) AcquireStaticProxySlot(ctx context.Context, accountID, staticProxyID int64, capacity int, requestID string) (bool, error) {
	f.acquireCalls = append(f.acquireCalls, staticProxyAcquireCall{
		accountID:     accountID,
		staticProxyID: staticProxyID,
		capacity:      capacity,
		requestID:     requestID,
	})
	if f.acquire != nil {
		return f.acquire(ctx, accountID, staticProxyID, capacity, requestID)
	}
	return false, nil
}

func (f *staticProxyConcurrencyCacheFake) ReleaseStaticProxySlot(ctx context.Context, accountID, staticProxyID int64, requestID string) error {
	f.releaseCalls = append(f.releaseCalls, staticProxyReleaseCall{
		accountID:     accountID,
		staticProxyID: staticProxyID,
		requestID:     requestID,
	})
	if f.release != nil {
		return f.release(ctx, accountID, staticProxyID, requestID)
	}
	return nil
}

func (f *staticProxyConcurrencyCacheFake) GetStaticProxyConcurrency(ctx context.Context, accountID, staticProxyID int64) (int, error) {
	if f.count != nil {
		return f.count(ctx, accountID, staticProxyID)
	}
	return 0, nil
}

func (f *staticProxyConcurrencyCacheFake) AcquireStaticProxyLiveSlot(context.Context, int64, int64, int, string) (bool, error) {
	return false, nil
}

func (f *staticProxyConcurrencyCacheFake) RefreshStaticProxyLiveSlot(ctx context.Context, accountID, staticProxyID int64, requestID string) (bool, error) {
	if f.refresh != nil {
		return f.refresh(ctx, accountID, staticProxyID, requestID)
	}
	return true, nil
}

func (f *staticProxyConcurrencyCacheFake) ReleaseStaticProxyLiveSlot(context.Context, int64, int64, string) error {
	return nil
}

func (f *staticProxyConcurrencyCacheFake) GetStaticProxyLiveConcurrency(context.Context, int64, int64) (int, error) {
	return 0, nil
}

var _ StaticProxyConcurrencyCache = (*staticProxyConcurrencyCacheFake)(nil)
var _ StaticProxyLiveConcurrencyCache = (*staticProxyConcurrencyCacheFake)(nil)

func TestStaticProxyAggregateCapacityDeduplicatesAssignments(t *testing.T) {
	require.Equal(t, 0, CalculateStaticProxyAggregateCapacity(nil))
	require.Equal(t, 10, CalculateStaticProxyAggregateCapacity([]int64{7, 7, 0, -1}))
	require.Equal(t, 30, CalculateStaticProxyAggregateCapacity([]int64{7, 8, 9}))
	require.Equal(t, StaticProxyCapacity, 10)
}

func TestConcurrencyServiceAcquireStaticProxyUsesAggregateCapacity(t *testing.T) {
	fake := &staticProxyConcurrencyCacheFake{
		acquire: func(_ context.Context, _ int64, staticProxyID int64, _ int, _ string) (bool, error) {
			return staticProxyID == 12, nil
		},
	}
	svc := NewConcurrencyService(fake)

	allocation, err := svc.AcquireStaticProxy(context.Background(), 42, []int64{11, 12, 12})
	require.NoError(t, err)
	require.NotNil(t, allocation)
	require.Equal(t, int64(42), allocation.AccountID)
	require.Equal(t, int64(12), allocation.StaticProxyID)
	require.NotEmpty(t, allocation.RequestID)
	require.NotNil(t, allocation.ReleaseFunc)
	require.Equal(t, []staticProxyAcquireCall{
		{accountID: 42, staticProxyID: 11, capacity: 20, requestID: allocation.RequestID},
		{accountID: 42, staticProxyID: 12, capacity: 20, requestID: allocation.RequestID},
	}, fake.acquireCalls)

	allocation.ReleaseFunc()
	require.Equal(t, []staticProxyReleaseCall{{accountID: 42, staticProxyID: 12, requestID: allocation.RequestID}}, fake.releaseCalls)
}

func TestConcurrencyServiceAcquireStaticProxyWithoutAssignmentsIsNoOp(t *testing.T) {
	fake := &staticProxyConcurrencyCacheFake{}
	allocation, err := NewConcurrencyService(fake).AcquireStaticProxy(context.Background(), 42, nil)
	require.NoError(t, err)
	require.Nil(t, allocation)
	require.Empty(t, fake.acquireCalls)
}

func TestConcurrencyServiceAcquireStaticProxyRequiresOptionalCache(t *testing.T) {
	allocation, err := NewConcurrencyService(&stubConcurrencyCacheForTest{}).AcquireStaticProxy(context.Background(), 42, []int64{11})
	require.Error(t, err)
	require.Nil(t, allocation)
}

func TestConcurrencyServiceAcquireStaticProxyReturnsCacheError(t *testing.T) {
	fake := &staticProxyConcurrencyCacheFake{
		acquire: func(context.Context, int64, int64, int, string) (bool, error) {
			return false, errors.New("redis down")
		},
	}

	allocation, err := NewConcurrencyService(fake).AcquireStaticProxy(context.Background(), 42, []int64{11})
	require.Error(t, err)
	require.Nil(t, allocation)
}
