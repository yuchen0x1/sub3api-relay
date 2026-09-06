package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/yuchen0x1/sub3api-relay/internal/pkg/errors"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/pagination"
)

const StaticProxyExitCapacity = 10

const (
	StaticProxyHealthUnknown   = "unknown"
	StaticProxyHealthHealthy   = "healthy"
	StaticProxyHealthDegraded  = "degraded"
	StaticProxyHealthUnhealthy = "unhealthy"
)

var (
	ErrStaticProxyExitNotFound = infraerrors.NotFound("STATIC_PROXY_EXIT_NOT_FOUND", "static proxy exit not found")
	ErrStaticProxyExitExists   = infraerrors.Conflict("STATIC_PROXY_EXIT_EXISTS", "static proxy exit already exists for this account and proxy")
)

// StaticProxyExitProxy is the credential-free proxy metadata exposed by static
// proxy management APIs. It intentionally has no username, password, or raw
// proxy URL fields.
type StaticProxyExitProxy struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Status   string `json:"status"`
}

// StaticProxyExit is an account-owned assignment of an existing proxy.
type StaticProxyExit struct {
	ID                int64                 `json:"id"`
	AccountID         int64                 `json:"account_id"`
	ProxyID           int64                 `json:"proxy_id"`
	Name              string                `json:"name"`
	Position          int                   `json:"position"`
	Enabled           bool                  `json:"enabled"`
	Capacity          int                   `json:"capacity"`
	HealthStatus      string                `json:"health_status"`
	FailureCount      int                   `json:"failure_count"`
	LastFailureAt     *time.Time            `json:"last_failure_at,omitempty"`
	LastFailureReason string                `json:"last_failure_reason,omitempty"`
	LastCheckedAt     *time.Time            `json:"last_checked_at,omitempty"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
	Proxy             *StaticProxyExitProxy `json:"proxy,omitempty"`
}

// StaticProxyAssignment is retained as a descriptive alias for callers that
// use assignment terminology.
type StaticProxyAssignment = StaticProxyExit

// StaticProxyExitListParams controls the admin list query.
type StaticProxyExitListParams struct {
	pagination.PaginationParams
	AccountID    *int64
	Enabled      *bool
	HealthStatus string
	Search       string
}

type StaticProxyExitCreateInput struct {
	AccountID int64
	ProxyID   int64
	Name      string
	Position  int
	Enabled   *bool
}

type StaticProxyExitUpdateInput struct {
	AccountID          *int64
	ProxyID            *int64
	Name               *string
	Position           *int
	Enabled            *bool
	HealthStatus       *string
	FailureCount       *int
	LastFailureAt      *time.Time
	LastFailureReason  *string
	LastCheckedAt      *time.Time
	ClearFailureAt     bool
	ClearFailureReason bool
	ClearCheckedAt     bool
}

type StaticProxyExitCapacitySummary struct {
	AccountID             *int64 `json:"account_id,omitempty"`
	AssignmentCount       int64  `json:"assignment_count"`
	EnabledCount          int64  `json:"enabled_count"`
	DisabledCount         int64  `json:"disabled_count"`
	CapacityPerAssignment int    `json:"capacity_per_assignment"`
	TotalCapacity         int64  `json:"total_capacity"`
	EnabledCapacity       int64  `json:"enabled_capacity"`
}

// StaticProxyRoutingTarget is an enabled assignment and its full existing proxy.
type StaticProxyRoutingTarget struct {
	ExitID  int64
	ProxyID int64
	Name    string
	Proxy   *Proxy
}

// StaticProxyRoutingLookup is an optional account runtime capability. Keeping
// it optional preserves legacy repositories/accounts and limits routing to the
// generic GatewayService.Forward path rather than all protocol handlers.
type StaticProxyRoutingLookup interface {
	ListEnabledRoutingTargets(ctx context.Context, accountID int64) ([]StaticProxyRoutingTarget, error)
}

type StaticProxyExitRepository interface {
	List(ctx context.Context, params StaticProxyExitListParams) ([]StaticProxyExit, int64, error)
	GetByID(ctx context.Context, id int64) (*StaticProxyExit, error)
	Create(ctx context.Context, exit *StaticProxyExit) error
	Update(ctx context.Context, exit *StaticProxyExit) error
	Delete(ctx context.Context, id int64) error
	GetCapacitySummary(ctx context.Context, accountID *int64) (*StaticProxyExitCapacitySummary, error)
}

// StaticProxyExitService owns validation and CRUD for account-owned proxy
// assignments. It does not participate in request concurrency accounting.
type StaticProxyExitService struct {
	repo        StaticProxyExitRepository
	accountRepo AccountRepository
	proxyRepo   ProxyRepository
}

func NewStaticProxyExitService(repo StaticProxyExitRepository, accountRepo AccountRepository, proxyRepo ProxyRepository) *StaticProxyExitService {
	return &StaticProxyExitService{repo: repo, accountRepo: accountRepo, proxyRepo: proxyRepo}
}

func (s *StaticProxyExitService) List(ctx context.Context, params StaticProxyExitListParams) ([]StaticProxyExit, int64, error) {
	return s.repo.List(ctx, normalizeStaticProxyExitListParams(params))
}

func (s *StaticProxyExitService) GetByID(ctx context.Context, id int64) (*StaticProxyExit, error) {
	if id <= 0 {
		return nil, ErrStaticProxyExitNotFound
	}
	return s.repo.GetByID(ctx, id)
}

func (s *StaticProxyExitService) Create(ctx context.Context, input StaticProxyExitCreateInput) (*StaticProxyExit, error) {
	input.Name = normalizeStaticProxyExitName(input.Name)
	// UI 将名称标为可选；留空时用代理名生成稳定默认名，而不是拒绝请求。
	if input.Name == "" {
		name, err := s.defaultExitName(ctx, input.ProxyID, input.Position)
		if err != nil {
			return nil, err
		}
		input.Name = name
	}
	if err := validateStaticProxyExitFields(input.AccountID, input.ProxyID, input.Name, input.Position); err != nil {
		return nil, err
	}
	if err := s.validateReferences(ctx, input.AccountID, input.ProxyID); err != nil {
		return nil, err
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	exit := &StaticProxyExit{
		AccountID:    input.AccountID,
		ProxyID:      input.ProxyID,
		Name:         input.Name,
		Position:     input.Position,
		Enabled:      enabled,
		Capacity:     StaticProxyExitCapacity,
		HealthStatus: StaticProxyHealthUnknown,
	}
	if err := s.repo.Create(ctx, exit); err != nil {
		return nil, err
	}
	return exit, nil
}

func (s *StaticProxyExitService) Update(ctx context.Context, id int64, input StaticProxyExitUpdateInput) (*StaticProxyExit, error) {
	exit, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.AccountID != nil {
		exit.AccountID = *input.AccountID
	}
	if input.ProxyID != nil {
		exit.ProxyID = *input.ProxyID
	}
	if input.Name != nil {
		exit.Name = normalizeStaticProxyExitName(*input.Name)
	}
	if input.Position != nil {
		exit.Position = *input.Position
	}
	if input.Enabled != nil {
		exit.Enabled = *input.Enabled
	}
	if input.HealthStatus != nil {
		exit.HealthStatus = normalizeStaticProxyHealthStatus(*input.HealthStatus)
	}
	if input.FailureCount != nil {
		exit.FailureCount = *input.FailureCount
	}
	if input.LastFailureAt != nil {
		exit.LastFailureAt = input.LastFailureAt
	}
	if input.LastFailureReason != nil {
		reason := normalizeStaticProxyFailureReason(*input.LastFailureReason)
		exit.LastFailureReason = reason
	}
	if input.LastCheckedAt != nil {
		exit.LastCheckedAt = input.LastCheckedAt
	}
	if input.ClearFailureAt {
		exit.LastFailureAt = nil
	}
	if input.ClearFailureReason {
		exit.LastFailureReason = ""
	}
	if input.ClearCheckedAt {
		exit.LastCheckedAt = nil
	}

	if err := validateStaticProxyExitFields(exit.AccountID, exit.ProxyID, exit.Name, exit.Position); err != nil {
		return nil, err
	}
	if err := validateStaticProxyHealthFields(exit.HealthStatus, exit.FailureCount); err != nil {
		return nil, err
	}
	if input.AccountID != nil || input.ProxyID != nil {
		if err := s.validateReferences(ctx, exit.AccountID, exit.ProxyID); err != nil {
			return nil, err
		}
	}
	exit.Capacity = StaticProxyExitCapacity
	if err := s.repo.Update(ctx, exit); err != nil {
		return nil, err
	}
	return exit, nil
}

func (s *StaticProxyExitService) Toggle(ctx context.Context, id int64, enabled *bool) (*StaticProxyExit, error) {
	exit, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if enabled == nil {
		exit.Enabled = !exit.Enabled
	} else {
		exit.Enabled = *enabled
	}
	if err := s.repo.Update(ctx, exit); err != nil {
		return nil, err
	}
	return exit, nil
}

func (s *StaticProxyExitService) Delete(ctx context.Context, id int64) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *StaticProxyExitService) GetCapacitySummary(ctx context.Context, accountID *int64) (*StaticProxyExitCapacitySummary, error) {
	if accountID != nil && *accountID <= 0 {
		return nil, infraerrors.BadRequest("STATIC_PROXY_ACCOUNT_INVALID", "account_id must be greater than zero")
	}
	return s.repo.GetCapacitySummary(ctx, accountID)
}

func (s *StaticProxyExitService) validateReferences(ctx context.Context, accountID, proxyID int64) error {
	if s.accountRepo != nil {
		if _, err := s.accountRepo.GetByID(ctx, accountID); err != nil {
			return err
		}
	}
	if s.proxyRepo != nil {
		if _, err := s.proxyRepo.GetByID(ctx, proxyID); err != nil {
			return err
		}
	}
	return nil
}

func normalizeStaticProxyExitListParams(params StaticProxyExitListParams) StaticProxyExitListParams {
	params.HealthStatus = normalizeStaticProxyHealthStatus(params.HealthStatus)
	params.Search = normalizeStaticProxyExitName(params.Search)
	return params
}

// defaultExitName derives a default exit name from the proxy when the caller
// leaves the name empty: "代理名-位置"（position > 0）或代理名本身。
func (s *StaticProxyExitService) defaultExitName(ctx context.Context, proxyID int64, position int) (string, error) {
	if s.proxyRepo == nil {
		return "", infraerrors.BadRequest("STATIC_PROXY_NAME_REQUIRED", "name is required")
	}
	p, err := s.proxyRepo.GetByID(ctx, proxyID)
	if err != nil {
		return "", err
	}
	base := normalizeStaticProxyExitName(p.Name)
	if base == "" {
		base = normalizeStaticProxyExitName(p.Host)
	}
	if base == "" {
		return "", infraerrors.BadRequest("STATIC_PROXY_NAME_REQUIRED", "name is required")
	}
	if position > 0 {
		return trimToMax(fmt.Sprintf("%s-%d", base, position), 100), nil
	}
	return base, nil
}

func normalizeStaticProxyExitName(name string) string {
	return trimToMax(name, 100)
}

func normalizeStaticProxyFailureReason(reason string) string {
	return trimToMax(reason, 2000)
}

func trimToMax(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) > max {
		return value[:max]
	}
	return value
}

func normalizeStaticProxyHealthStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func validateStaticProxyExitFields(accountID, proxyID int64, name string, position int) error {
	if accountID <= 0 {
		return infraerrors.BadRequest("STATIC_PROXY_ACCOUNT_INVALID", "account_id must be greater than zero")
	}
	if proxyID <= 0 {
		return infraerrors.BadRequest("STATIC_PROXY_PROXY_INVALID", "proxy_id must be greater than zero")
	}
	if name == "" {
		return infraerrors.BadRequest("STATIC_PROXY_NAME_REQUIRED", "name is required")
	}
	if position < 0 {
		return infraerrors.BadRequest("STATIC_PROXY_POSITION_INVALID", "position must be greater than or equal to zero")
	}
	return nil
}

func validateStaticProxyHealthFields(status string, failureCount int) error {
	switch normalizeStaticProxyHealthStatus(status) {
	case StaticProxyHealthUnknown, StaticProxyHealthHealthy, StaticProxyHealthDegraded, StaticProxyHealthUnhealthy:
	default:
		return infraerrors.BadRequest("STATIC_PROXY_HEALTH_STATUS_INVALID", "health_status must be unknown, healthy, degraded, or unhealthy")
	}
	if failureCount < 0 {
		return infraerrors.BadRequest("STATIC_PROXY_FAILURE_COUNT_INVALID", "failure_count must be greater than or equal to zero")
	}
	return nil
}
