package admin

import (
	"time"

	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

// StaticProxyExitProxyDTO is the credential-free proxy metadata exposed by the
// static proxy exit API. It intentionally has no username, password, or raw
// proxy URL fields so that proxy credentials are never leaked to the panel.
type StaticProxyExitProxyDTO struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Status   string `json:"status"`
}

// StaticProxyExitDTO is the API representation of an account-owned assignment
// of an existing proxy.
type StaticProxyExitDTO struct {
	ID                int64                    `json:"id"`
	AccountID         int64                    `json:"account_id"`
	ProxyID           int64                    `json:"proxy_id"`
	Name              string                   `json:"name"`
	Position          int                      `json:"position"`
	Enabled           bool                     `json:"enabled"`
	Capacity          int                      `json:"capacity"`
	HealthStatus      string                   `json:"health_status"`
	FailureCount      int                      `json:"failure_count"`
	LastFailureAt     *time.Time               `json:"last_failure_at,omitempty"`
	LastFailureReason string                   `json:"last_failure_reason,omitempty"`
	LastCheckedAt     *time.Time               `json:"last_checked_at,omitempty"`
	CreatedAt         time.Time                `json:"created_at"`
	UpdatedAt         time.Time                `json:"updated_at"`
	Proxy             *StaticProxyExitProxyDTO `json:"proxy,omitempty"`
}

// StaticProxyExitCapacitySummaryDTO summarizes assignment capacity for the
// whole system or a single account.
type StaticProxyExitCapacitySummaryDTO struct {
	AccountID             *int64 `json:"account_id,omitempty"`
	AssignmentCount       int64  `json:"assignment_count"`
	EnabledCount          int64  `json:"enabled_count"`
	DisabledCount         int64  `json:"disabled_count"`
	CapacityPerAssignment int    `json:"capacity_per_assignment"`
	TotalCapacity         int64  `json:"total_capacity"`
	EnabledCapacity       int64  `json:"enabled_capacity"`
}

// StaticProxyExitCreateRequest is the create payload for a static proxy exit.
type StaticProxyExitCreateRequest struct {
	AccountID int64  `json:"account_id" binding:"required"`
	ProxyID   int64  `json:"proxy_id" binding:"required"`
	Name      string `json:"name"`
	Position  int    `json:"position"`
	Enabled   *bool  `json:"enabled"`
}

// StaticProxyExitUpdateRequest is the update payload for a static proxy exit.
type StaticProxyExitUpdateRequest struct {
	AccountID *int64  `json:"account_id"`
	ProxyID   *int64  `json:"proxy_id"`
	Name      *string `json:"name"`
	Position  *int    `json:"position"`
	Enabled   *bool   `json:"enabled"`
}

// StaticProxyExitToggleRequest toggles the enabled flag. When enabled is
// omitted the current value is inverted.
type StaticProxyExitToggleRequest struct {
	Enabled *bool `json:"enabled"`
}

func staticProxyExitProxyFromService(p *service.StaticProxyExitProxy) *StaticProxyExitProxyDTO {
	if p == nil {
		return nil
	}
	return &StaticProxyExitProxyDTO{
		ID:       p.ID,
		Name:     p.Name,
		Protocol: p.Protocol,
		Host:     p.Host,
		Port:     p.Port,
		Status:   p.Status,
	}
}

func staticProxyExitFromService(e *service.StaticProxyExit) *StaticProxyExitDTO {
	if e == nil {
		return nil
	}
	return &StaticProxyExitDTO{
		ID:                e.ID,
		AccountID:         e.AccountID,
		ProxyID:           e.ProxyID,
		Name:              e.Name,
		Position:          e.Position,
		Enabled:           e.Enabled,
		Capacity:          e.Capacity,
		HealthStatus:      e.HealthStatus,
		FailureCount:      e.FailureCount,
		LastFailureAt:     e.LastFailureAt,
		LastFailureReason: e.LastFailureReason,
		LastCheckedAt:     e.LastCheckedAt,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
		Proxy:             staticProxyExitProxyFromService(e.Proxy),
	}
}

func staticProxyExitsFromService(items []service.StaticProxyExit) []StaticProxyExitDTO {
	out := make([]StaticProxyExitDTO, 0, len(items))
	for i := range items {
		if d := staticProxyExitFromService(&items[i]); d != nil {
			out = append(out, *d)
		}
	}
	return out
}

func staticProxyExitCapacitySummaryFromService(s *service.StaticProxyExitCapacitySummary) *StaticProxyExitCapacitySummaryDTO {
	if s == nil {
		return nil
	}
	return &StaticProxyExitCapacitySummaryDTO{
		AccountID:             s.AccountID,
		AssignmentCount:       s.AssignmentCount,
		EnabledCount:          s.EnabledCount,
		DisabledCount:         s.DisabledCount,
		CapacityPerAssignment: s.CapacityPerAssignment,
		TotalCapacity:         s.TotalCapacity,
		EnabledCapacity:       s.EnabledCapacity,
	}
}
