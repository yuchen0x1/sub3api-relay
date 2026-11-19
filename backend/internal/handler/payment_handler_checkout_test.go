//go:build unit

package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	dbent "github.com/yuchen0x1/sub3api-relay/ent"
	"github.com/yuchen0x1/sub3api-relay/ent/enttest"
	"github.com/yuchen0x1/sub3api-relay/ent/subscriptionplan"
	"github.com/yuchen0x1/sub3api-relay/internal/repository"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
	_ "modernc.org/sqlite"
)

func TestGetCheckoutInfoReturnsErrorWhenPlanQueryFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite", "file:checkout_info_plan_error?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	_, err = db.Exec("DROP TABLE " + subscriptionplan.Table)
	require.NoError(t, err)

	configService := service.NewPaymentConfigService(client, repository.NewSettingRepository(client), nil)
	handler := NewPaymentHandler(nil, configService)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/checkout-info", nil)

	handler.GetCheckoutInfo(ctx)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}
