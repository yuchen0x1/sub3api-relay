package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

func TestOpsRepositoryQueryErrorCounts_ExcludesClientClosedRequests(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}

	start := time.Date(2026, 8, 10, 2, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	filter := &service.OpsDashboardFilter{StartTime: start, EndTime: end}

	mock.ExpectQuery(`(?s)NOT \(error_owner = 'client' AND COALESCE\(status_code, 0\) = 499\)`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"error_total", "business_limited", "error_sla",
			"upstream_excl", "upstream_429", "upstream_529",
		}).AddRow(int64(31), int64(0), int64(0), int64(0), int64(0), int64(0)))

	errorTotal, businessLimited, errorSLA, upstreamExcl429529, upstream429, upstream529, err :=
		repo.queryErrorCounts(context.Background(), filter, start, end)
	require.NoError(t, err)
	require.Equal(t, int64(31), errorTotal)
	require.Zero(t, businessLimited)
	require.Zero(t, errorSLA)
	require.Zero(t, upstreamExcl429529)
	require.Zero(t, upstream429)
	require.Zero(t, upstream529)
}

func TestOpsRepositoryUpsertHourlyMetrics_ExcludesClientClosedRequests(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}

	start := time.Date(2026, 8, 10, 2, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	mock.ExpectExec(`(?s)NOT \(error_owner = 'client' AND COALESCE\(client_status_code, 0\) = 499\)`).
		WithArgs(start, end).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpsertHourlyMetrics(context.Background(), start, end))
}

func TestOpsRepositoryGetErrorTrend_ExcludesClientClosedRequests(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}

	start := time.Date(2026, 8, 10, 2, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	filter := &service.OpsDashboardFilter{StartTime: start, EndTime: end}

	mock.ExpectQuery(`(?s)NOT \(error_owner = 'client' AND COALESCE\(status_code, 0\) = 499\)`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"bucket", "error_total", "business_limited", "error_sla",
			"upstream_excl", "upstream_429", "upstream_529",
		}))

	_, err := repo.GetErrorTrend(context.Background(), filter, 60)
	require.NoError(t, err)
}
