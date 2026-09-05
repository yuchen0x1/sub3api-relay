package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

func TestOpsRepositoryListRequestDetailsSortsByFirstToken(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &opsRepository{db: db}

	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	filter := &service.OpsRequestDetailFilter{
		StartTime: &start,
		EndTime:   &end,
		Sort:      "first_token_desc",
		Page:      1,
		PageSize:  10,
	}

	mock.ExpectQuery(`SELECT COUNT\(1\) FROM combined`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	createdAt := start.Add(30 * time.Minute)
	mock.ExpectQuery(`ORDER BY first_token_ms DESC NULLS LAST, created_at DESC\s+LIMIT \$3 OFFSET \$4`).
		WithArgs(start, end, 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"kind", "created_at", "request_id", "platform", "model", "duration_ms", "first_token_ms",
			"status_code", "error_id", "phase", "severity", "message", "user_id", "api_key_id",
			"account_id", "group_id", "stream",
		}).AddRow(
			"success", createdAt, "req-ttft", "openai", "gpt-5", 30_000, 26_461,
			nil, nil, nil, nil, nil, int64(1), int64(2), int64(3), int64(4), true,
		))

	items, total, err := repo.ListRequestDetails(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].FirstTokenMs)
	require.Equal(t, 26_461, *items[0].FirstTokenMs)
	require.NoError(t, mock.ExpectationsWereMet())
}
