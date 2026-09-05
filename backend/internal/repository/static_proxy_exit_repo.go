package repository

import (
	"context"
	"strings"

	dbent "github.com/yuchen0x1/sub3api-relay/ent"
	"github.com/yuchen0x1/sub3api-relay/ent/staticproxy"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/pagination"
	"github.com/yuchen0x1/sub3api-relay/internal/service"

	entsql "entgo.io/ent/dialect/sql"
)

type staticProxyExitRepository struct {
	client *dbent.Client
}

// NewStaticProxyExitRepository creates the concrete repository for account-owned
// static proxy exits backed by the Ent schema.
func NewStaticProxyExitRepository(client *dbent.Client) service.StaticProxyExitRepository {
	return &staticProxyExitRepository{client: client}
}

func (r *staticProxyExitRepository) List(ctx context.Context, params service.StaticProxyExitListParams) ([]service.StaticProxyExit, int64, error) {
	client := clientFromContext(ctx, r.client)
	q := client.StaticProxy.Query()

	if params.AccountID != nil {
		q = q.Where(staticproxy.AccountIDEQ(*params.AccountID))
	}
	if params.Enabled != nil {
		q = q.Where(staticproxy.EnabledEQ(*params.Enabled))
	}
	if params.HealthStatus != "" {
		q = q.Where(staticproxy.HealthStatusEQ(params.HealthStatus))
	}
	if params.Search != "" {
		q = q.Where(staticproxy.NameContainsFold(params.Search))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	itemsQuery := q.
		WithProxy().
		Offset(params.Offset()).
		Limit(params.Limit())
	for _, order := range staticProxyExitListOrders(params.PaginationParams) {
		itemsQuery = itemsQuery.Order(order)
	}

	items, err := itemsQuery.All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return staticProxyExitEntitiesToService(items), int64(total), nil
}

func (r *staticProxyExitRepository) ListEnabledRoutingTargets(ctx context.Context, accountID int64) ([]service.StaticProxyRoutingTarget, error) {
	if accountID <= 0 {
		return nil, nil
	}
	client := clientFromContext(ctx, r.client)
	items, err := client.StaticProxy.Query().
		Where(staticproxy.AccountIDEQ(accountID), staticproxy.EnabledEQ(true)).
		WithProxy().
		Order(staticproxy.ByPosition(), staticproxy.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	targets := make([]service.StaticProxyRoutingTarget, 0, len(items))
	for _, item := range items {
		if item == nil || item.Edges.Proxy == nil {
			continue
		}
		targets = append(targets, service.StaticProxyRoutingTarget{
			ExitID:  item.ID,
			ProxyID: item.ProxyID,
			Name:    item.Name,
			Proxy:   proxyEntityToService(item.Edges.Proxy),
		})
	}
	return targets, nil
}

func (r *staticProxyExitRepository) GetByID(ctx context.Context, id int64) (*service.StaticProxyExit, error) {
	m, err := r.client.StaticProxy.Query().
		Where(staticproxy.IDEQ(id)).
		WithProxy().
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrStaticProxyExitNotFound, nil)
	}
	return staticProxyExitEntityToService(m), nil
}

func (r *staticProxyExitRepository) Create(ctx context.Context, exit *service.StaticProxyExit) error {
	client := clientFromContext(ctx, r.client)
	builder := client.StaticProxy.Create().
		SetAccountID(exit.AccountID).
		SetProxyID(exit.ProxyID).
		SetName(exit.Name).
		SetPosition(exit.Position).
		SetEnabled(exit.Enabled).
		SetCapacity(exit.Capacity).
		SetHealthStatus(exit.HealthStatus).
		SetFailureCount(exit.FailureCount)

	if exit.LastFailureAt != nil {
		builder.SetLastFailureAt(*exit.LastFailureAt)
	}
	if exit.LastFailureReason != "" {
		builder.SetLastFailureReason(exit.LastFailureReason)
	}
	if exit.LastCheckedAt != nil {
		builder.SetLastCheckedAt(*exit.LastCheckedAt)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, nil, service.ErrStaticProxyExitExists)
	}
	applyStaticProxyExitEntityToService(exit, created)
	return nil
}

func (r *staticProxyExitRepository) Update(ctx context.Context, exit *service.StaticProxyExit) error {
	client := clientFromContext(ctx, r.client)
	builder := client.StaticProxy.UpdateOneID(exit.ID).
		SetAccountID(exit.AccountID).
		SetProxyID(exit.ProxyID).
		SetName(exit.Name).
		SetPosition(exit.Position).
		SetEnabled(exit.Enabled).
		SetCapacity(exit.Capacity).
		SetHealthStatus(exit.HealthStatus).
		SetFailureCount(exit.FailureCount)

	if exit.LastFailureAt != nil {
		builder.SetLastFailureAt(*exit.LastFailureAt)
	} else {
		builder.ClearLastFailureAt()
	}
	if exit.LastFailureReason != "" {
		builder.SetLastFailureReason(exit.LastFailureReason)
	} else {
		builder.ClearLastFailureReason()
	}
	if exit.LastCheckedAt != nil {
		builder.SetLastCheckedAt(*exit.LastCheckedAt)
	} else {
		builder.ClearLastCheckedAt()
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrStaticProxyExitNotFound, service.ErrStaticProxyExitExists)
	}
	exit.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *staticProxyExitRepository) Delete(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.StaticProxy.Delete().Where(staticproxy.IDEQ(id)).Exec(ctx)
	return translatePersistenceError(err, service.ErrStaticProxyExitNotFound, nil)
}

func (r *staticProxyExitRepository) GetCapacitySummary(ctx context.Context, accountID *int64) (*service.StaticProxyExitCapacitySummary, error) {
	client := clientFromContext(ctx, r.client)

	q := client.StaticProxy.Query()
	if accountID != nil {
		q = q.Where(staticproxy.AccountIDEQ(*accountID))
	}

	assignmentCount, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}

	enabledQ := client.StaticProxy.Query().Where(staticproxy.EnabledEQ(true))
	if accountID != nil {
		enabledQ = enabledQ.Where(staticproxy.AccountIDEQ(*accountID))
	}
	enabledCount, err := enabledQ.Count(ctx)
	if err != nil {
		return nil, err
	}

	capacity := int64(service.StaticProxyExitCapacity)
	return &service.StaticProxyExitCapacitySummary{
		AccountID:             accountID,
		AssignmentCount:       int64(assignmentCount),
		EnabledCount:          int64(enabledCount),
		DisabledCount:         int64(assignmentCount - enabledCount),
		CapacityPerAssignment: service.StaticProxyExitCapacity,
		TotalCapacity:         int64(assignmentCount) * capacity,
		EnabledCapacity:       int64(enabledCount) * capacity,
	}, nil
}

func staticProxyExitListOrders(params pagination.PaginationParams) []func(*entsql.Selector) {
	field, order := staticProxyExitListOrder(params)
	if order == pagination.SortOrderAsc {
		if field == staticproxy.FieldID {
			return []func(*entsql.Selector){dbent.Asc(field)}
		}
		return []func(*entsql.Selector){dbent.Asc(field), dbent.Asc(staticproxy.FieldID)}
	}
	if field == staticproxy.FieldID {
		return []func(*entsql.Selector){dbent.Desc(field)}
	}
	return []func(*entsql.Selector){dbent.Desc(field), dbent.Desc(staticproxy.FieldID)}
}

func staticProxyExitListOrder(params pagination.PaginationParams) (string, string) {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)

	switch sortBy {
	case "account_id":
		return staticproxy.FieldAccountID, sortOrder
	case "proxy_id":
		return staticproxy.FieldProxyID, sortOrder
	case "name":
		return staticproxy.FieldName, sortOrder
	case "position":
		return staticproxy.FieldPosition, sortOrder
	case "enabled":
		return staticproxy.FieldEnabled, sortOrder
	case "capacity":
		return staticproxy.FieldCapacity, sortOrder
	case "health_status":
		return staticproxy.FieldHealthStatus, sortOrder
	case "failure_count":
		return staticproxy.FieldFailureCount, sortOrder
	case "id":
		return staticproxy.FieldID, sortOrder
	case "", "created_at":
		return staticproxy.FieldCreatedAt, sortOrder
	default:
		return staticproxy.FieldCreatedAt, pagination.SortOrderDesc
	}
}

func applyStaticProxyExitEntityToService(dst *service.StaticProxyExit, src *dbent.StaticProxy) {
	if dst == nil || src == nil {
		return
	}
	dst.ID = src.ID
	dst.CreatedAt = src.CreatedAt
	dst.UpdatedAt = src.UpdatedAt
}

func staticProxyExitEntityToService(m *dbent.StaticProxy) *service.StaticProxyExit {
	if m == nil {
		return nil
	}
	out := &service.StaticProxyExit{
		ID:                m.ID,
		AccountID:         m.AccountID,
		ProxyID:           m.ProxyID,
		Name:              m.Name,
		Position:          m.Position,
		Enabled:           m.Enabled,
		Capacity:          m.Capacity,
		HealthStatus:      m.HealthStatus,
		FailureCount:      m.FailureCount,
		LastFailureAt:     m.LastFailureAt,
		LastFailureReason: staticProxyExitDerefString(m.LastFailureReason),
		LastCheckedAt:     m.LastCheckedAt,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
	if m.Edges.Proxy != nil {
		out.Proxy = &service.StaticProxyExitProxy{
			ID:       m.Edges.Proxy.ID,
			Name:     m.Edges.Proxy.Name,
			Protocol: m.Edges.Proxy.Protocol,
			Host:     m.Edges.Proxy.Host,
			Port:     m.Edges.Proxy.Port,
			Status:   m.Edges.Proxy.Status,
		}
	}
	return out
}

func staticProxyExitEntitiesToService(models []*dbent.StaticProxy) []service.StaticProxyExit {
	out := make([]service.StaticProxyExit, 0, len(models))
	for i := range models {
		if s := staticProxyExitEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}

func staticProxyExitDerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
