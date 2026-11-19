package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yuchen0x1/sub3api-relay/internal/payment"
)

func TestCheckDailyLimitReservesPendingBalanceOrders(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	user, err := client.User.Create().
		SetEmail("daily-limit@example.com").
		SetPasswordHash("hash").
		SetUsername("daily-limit-user").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(70).
		SetPayAmount(80).
		SetFeeRate(0).
		SetRechargeCode("PAY-PENDING-1").
		SetOutTradeNo("pending-daily-limit-1").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusPending).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("app.example.com").
		Save(ctx)
	require.NoError(t, err)

	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })

	svc := &PaymentService{entClient: client}
	require.Error(t, svc.checkDailyLimit(ctx, tx, user.ID, 21, 100))
	require.NoError(t, svc.checkDailyLimit(ctx, tx, user.ID, 20, 100))
}

func TestCheckPendingLimitRejectsAtConfiguredMaximum(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	user, err := client.User.Create().
		SetEmail("pending-limit@example.com").
		SetPasswordHash("hash").
		SetUsername("pending-limit-user").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(10).
		SetFeeRate(0).
		SetRechargeCode("PAY-PENDING-2").
		SetOutTradeNo("pending-limit-1").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusPending).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("app.example.com").
		Save(ctx)
	require.NoError(t, err)

	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })

	svc := &PaymentService{entClient: client}
	require.Error(t, svc.checkPendingLimit(ctx, tx, user.ID, 1))
}
