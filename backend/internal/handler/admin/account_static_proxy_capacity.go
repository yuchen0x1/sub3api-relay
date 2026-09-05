package admin

import (
	"context"
	"log/slog"

	"github.com/yuchen0x1/sub3api-relay/internal/pkg/pagination"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

// staticProxyStatusDisabled is the per-exit status reported for assignments
// that are disabled, matching the frontend StaticProxyCapacity badge styling.
const staticProxyStatusDisabled = "disabled"

// staticProxyExitLister lists account-owned static proxy exit assignments. It
// is deliberately minimal so admin account responses only depend on listing and
// a full repository or service is not required. *service.StaticProxyExitService
// satisfies this interface. It stays optional so legacy deployments and test
// doubles without static proxy capacity are unaffected.
type staticProxyExitLister interface {
	List(ctx context.Context, params service.StaticProxyExitListParams) ([]service.StaticProxyExit, int64, error)
}

// AccountStaticProxyExitCapacity is one account-owned static proxy assignment
// with its live Redis concurrency count.
type AccountStaticProxyExitCapacity struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Host    string `json:"host"`
	Status  string `json:"status"`
	Current int    `json:"current"`
	Max     int    `json:"max"`
}

// AccountStaticIPCapacity aggregates account-wide static proxy capacity. Max
// reflects usable capacity (enabled assignments), while Current is the sum of
// the live per-assignment Redis concurrency counts.
type AccountStaticIPCapacity struct {
	Current int                              `json:"current"`
	Max     int                              `json:"max"`
	Exits   []AccountStaticProxyExitCapacity `json:"exits"`
}

// SetStaticProxyExitLister attaches the optional static proxy exit lister used
// to enrich admin account responses. nil keeps the legacy response shape.
func (h *AccountHandler) SetStaticProxyExitLister(lister staticProxyExitLister) {
	h.staticProxyExitLister = lister
}

// buildAccountStaticProxyCapacity computes account-wide static proxy capacity
// from the account's static proxy exit assignments and live Redis concurrency
// counts. It returns nil when the optional lister is missing or the account has
// no assignments so legacy responses are preserved byte-for-byte.
func (h *AccountHandler) buildAccountStaticProxyCapacity(ctx context.Context, account *service.Account) *AccountStaticIPCapacity {
	if h == nil || h.staticProxyExitLister == nil || account == nil {
		return nil
	}
	accountID := account.ID
	exits, _, err := h.staticProxyExitLister.List(ctx, service.StaticProxyExitListParams{
		PaginationParams: pagination.PaginationParams{
			Page:      1,
			PageSize:  1000,
			SortBy:    "position",
			SortOrder: pagination.SortOrderAsc,
		},
		AccountID: &accountID,
	})
	if err != nil {
		slog.Warn("account_static_proxy_capacity_list_failed", "account_id", accountID, "error", err)
		return nil
	}
	if len(exits) == 0 {
		return nil
	}

	capacity := &AccountStaticIPCapacity{
		Exits: make([]AccountStaticProxyExitCapacity, 0, len(exits)),
	}
	var aggregateCurrent, aggregateMax int
	for i := range exits {
		exit := &exits[i]

		current := 0
		if h.concurrencyService != nil {
			if count, countErr := h.concurrencyService.GetStaticProxyConcurrency(ctx, accountID, exit.ID); countErr == nil {
				current = count
			}
		}

		max := exit.Capacity
		if max <= 0 {
			max = service.StaticProxyExitCapacity
		}

		status := exit.HealthStatus
		if !exit.Enabled {
			status = staticProxyStatusDisabled
		}

		host := ""
		if exit.Proxy != nil {
			host = exit.Proxy.Host
		}

		aggregateCurrent += current
		if exit.Enabled {
			aggregateMax += max
		}

		capacity.Exits = append(capacity.Exits, AccountStaticProxyExitCapacity{
			ID:      exit.ID,
			Name:    exit.Name,
			Host:    host,
			Status:  status,
			Current: current,
			Max:     max,
		})
	}
	capacity.Current = aggregateCurrent
	capacity.Max = aggregateMax
	return capacity
}
