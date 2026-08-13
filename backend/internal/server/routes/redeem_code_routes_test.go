package routes

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/handler"
	adminhandler "github.com/yuchen0x1/sub3api-relay/internal/handler/admin"
	servermiddleware "github.com/yuchen0x1/sub3api-relay/internal/server/middleware"
)

func TestRedeemCodeAdminRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{
		Redeem: adminhandler.NewRedeemHandler(nil, nil),
	}}
	pass := func(c *gin.Context) { c.Next() }

	RegisterAdminRoutes(
		router.Group("/api/v1"),
		handlers,
		servermiddleware.AdminAuthMiddleware(pass),
		servermiddleware.AuditLogMiddleware(pass),
		servermiddleware.StepUpAuthMiddleware(pass),
		nil,
		nil,
	)

	registered := make(map[string]bool)
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	for _, route := range []string{
		"GET /api/v1/admin/redeem-codes",
		"GET /api/v1/admin/redeem-codes/stats",
		"GET /api/v1/admin/redeem-codes/export",
		"POST /api/v1/admin/redeem-codes/generate",
		"POST /api/v1/admin/redeem-codes/batch-update",
	} {
		require.Truef(t, registered[route], "%s should be registered", route)
	}
}
