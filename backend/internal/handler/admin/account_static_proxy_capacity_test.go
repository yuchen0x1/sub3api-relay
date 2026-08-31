package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

// staticProxyCapacityTestCache stubs the ConcurrencyCache surface used by the
// account handler list/detail enrichment paths. The embedded ConcurrencyCache
// is never called on these paths, so it can remain nil.
type staticProxyCapacityTestCache struct {
	service.ConcurrencyCache
	accountCounts     map[int64]int
	staticProxyCounts map[int64]int
}

func (c *staticProxyCapacityTestCache) GetAccountConcurrencyBatch(_ context.Context, accountIDs []int64) (map[int64]int, error) {
	out := make(map[int64]int, len(accountIDs))
	for _, id := range accountIDs {
		out[id] = c.accountCounts[id]
	}
	return out, nil
}

func (c *staticProxyCapacityTestCache) GetStaticProxyConcurrency(_ context.Context, _, staticProxyID int64) (int, error) {
	return c.staticProxyCounts[staticProxyID], nil
}

// The remaining StaticProxyConcurrencyCache members are required for the type
// assertion in ConcurrencyService.GetStaticProxyConcurrency but are not used by
// the read-only enrichment path.
func (c *staticProxyCapacityTestCache) AcquireStaticProxySlot(context.Context, int64, int64, int, string) (bool, error) {
	return true, nil
}

func (c *staticProxyCapacityTestCache) ReleaseStaticProxySlot(context.Context, int64, int64, string) error {
	return nil
}

// staticProxyExitListerFake records the list params and serves canned exits.
type staticProxyExitListerFake struct {
	exits      []service.StaticProxyExit
	err        error
	lastParams service.StaticProxyExitListParams
}

func (f *staticProxyExitListerFake) List(_ context.Context, params service.StaticProxyExitListParams) ([]service.StaticProxyExit, int64, error) {
	f.lastParams = params
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.exits, int64(len(f.exits)), nil
}

func sampleStaticProxyExits() []service.StaticProxyExit {
	return []service.StaticProxyExit{
		{
			ID: 11, AccountID: 3, ProxyID: 4, Name: "exit-a", Position: 0, Enabled: true,
			Capacity: 10, HealthStatus: service.StaticProxyHealthHealthy,
			Proxy: &service.StaticProxyExitProxy{ID: 4, Protocol: "http", Host: "one.example.com", Port: 8080},
		},
		{
			ID: 12, AccountID: 3, ProxyID: 5, Name: "exit-b", Position: 1, Enabled: true,
			Capacity: 10, HealthStatus: service.StaticProxyHealthDegraded,
			Proxy: &service.StaticProxyExitProxy{ID: 5, Protocol: "socks5", Host: "two.example.com", Port: 1080},
		},
		{
			ID: 13, AccountID: 3, ProxyID: 6, Name: "exit-c", Position: 2, Enabled: false,
			Capacity: 10, HealthStatus: service.StaticProxyHealthHealthy,
			Proxy: &service.StaticProxyExitProxy{ID: 6, Protocol: "http", Host: "three.example.com"},
		},
	}
}

func newStaticProxyCapacityTestHandler() (*AccountHandler, *staticProxyExitListerFake) {
	adminSvc := newStubAdminService()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	lister := &staticProxyExitListerFake{exits: sampleStaticProxyExits()}
	handler.SetStaticProxyExitLister(lister)
	handler.concurrencyService = service.NewConcurrencyService(&staticProxyCapacityTestCache{
		accountCounts:     map[int64]int{3: 7},
		staticProxyCounts: map[int64]int{11: 4, 12: 3, 13: 0},
	})
	return handler, lister
}

func TestAccountStaticProxyCapacity_ListEnrichment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, lister := newStaticProxyCapacityTestHandler()

	router := gin.New()
	router.GET("/api/v1/admin/accounts", handler.List)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts?page=1&page_size=20", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// The lister must be scoped to the account and request all assignments.
	require.Equal(t, int64(3), *lister.lastParams.AccountID)
	require.Equal(t, 1000, lister.lastParams.PageSize)
	require.Equal(t, "position", lister.lastParams.SortBy)

	var payload struct {
		Data struct {
			Items []struct {
				ID               int64 `json:"id"`
				StaticIPCapacity *struct {
					Current int `json:"current"`
					Max     int `json:"max"`
					Exits   []struct {
						ID      int64  `json:"id"`
						Name    string `json:"name"`
						Host    string `json:"host"`
						Status  string `json:"status"`
						Current int    `json:"current"`
						Max     int    `json:"max"`
					} `json:"exits"`
				} `json:"static_ip_capacity"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Len(t, payload.Data.Items, 1)
	require.Equal(t, int64(3), payload.Data.Items[0].ID)

	capacity := payload.Data.Items[0].StaticIPCapacity
	require.NotNil(t, capacity)
	require.Equal(t, 7, capacity.Current, "aggregate current sums live Redis counts")
	require.Equal(t, 20, capacity.Max, "aggregate max only counts enabled assignments")
	require.Len(t, capacity.Exits, 3)

	require.Equal(t, int64(11), capacity.Exits[0].ID)
	require.Equal(t, "exit-a", capacity.Exits[0].Name)
	require.Equal(t, "one.example.com", capacity.Exits[0].Host)
	require.Equal(t, service.StaticProxyHealthHealthy, capacity.Exits[0].Status)
	require.Equal(t, 4, capacity.Exits[0].Current)
	require.Equal(t, 10, capacity.Exits[0].Max)

	require.Equal(t, "two.example.com", capacity.Exits[1].Host)
	require.Equal(t, service.StaticProxyHealthDegraded, capacity.Exits[1].Status)
	require.Equal(t, 3, capacity.Exits[1].Current)

	require.Equal(t, int64(13), capacity.Exits[2].ID)
	require.Equal(t, "three.example.com", capacity.Exits[2].Host)
	require.Equal(t, staticProxyStatusDisabled, capacity.Exits[2].Status)
	require.Equal(t, 0, capacity.Exits[2].Current)
}

func TestAccountStaticProxyCapacity_ListOmitsFieldWithoutLister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	router := gin.New()
	router.GET("/api/v1/admin/accounts", handler.List)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts?page=1&page_size=20", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var payload struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Len(t, payload.Data.Items, 1)
	require.NotContains(t, payload.Data.Items[0], "static_ip_capacity")
}

func TestAccountStaticProxyCapacity_DetailEnrichment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, lister := newStaticProxyCapacityTestHandler()

	router := gin.New()
	router.GET("/api/v1/admin/accounts/:id", handler.GetByID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/3", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var payload struct {
		Data struct {
			ID               int64 `json:"id"`
			StaticIPCapacity *struct {
				Current int              `json:"current"`
				Max     int              `json:"max"`
				Exits   []map[string]any `json:"exits"`
			} `json:"static_ip_capacity"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, int64(3), payload.Data.ID)
	require.NotNil(t, payload.Data.StaticIPCapacity)
	require.Equal(t, 7, payload.Data.StaticIPCapacity.Current)
	require.Equal(t, 20, payload.Data.StaticIPCapacity.Max)
	require.Equal(t, 1000, lister.lastParams.PageSize, "detail paths must request all assignments")
}

func TestAccountStaticProxyCapacity_ListerErrorIsGraceful(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.SetStaticProxyExitLister(&staticProxyExitListerFake{err: errors.New("db down")})

	router := gin.New()
	router.GET("/api/v1/admin/accounts/:id", handler.GetByID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/3", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var payload struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.NotContains(t, payload.Data, "static_ip_capacity")
	require.NotNil(t, payload.Data["id"])
}

func TestAccountStaticProxyCapacity_ZeroCurrentWithoutConcurrencyService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	lister := &staticProxyExitListerFake{exits: sampleStaticProxyExits()}
	handler.SetStaticProxyExitLister(lister)
	// No concurrency service: per-exit current must degrade to 0 while max is kept.

	router := gin.New()
	router.GET("/api/v1/admin/accounts/:id", handler.GetByID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/3", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var payload struct {
		Data struct {
			StaticIPCapacity *struct {
				Current int `json:"current"`
				Max     int `json:"max"`
			} `json:"static_ip_capacity"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.NotNil(t, payload.Data.StaticIPCapacity)
	require.Equal(t, 0, payload.Data.StaticIPCapacity.Current)
	require.Equal(t, 20, payload.Data.StaticIPCapacity.Max)
}
