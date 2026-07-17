package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertUserForPaymentGuardTest(t *testing.T, id int, quota int) {
	t.Helper()
	user := &User{
		Id:       id,
		Username: "payment_guard_user",
		Status:   common.UserStatusEnabled,
		Quota:    quota,
	}
	require.NoError(t, DB.Create(user).Error)
}

func insertSubscriptionPlanForPaymentGuardTest(t *testing.T, id int) *SubscriptionPlan {
	t.Helper()
	plan := &SubscriptionPlan{
		Id:            id,
		Title:         "Guard Plan",
		PriceAmount:   9.99,
		Currency:      "USD",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   1000,
	}
	require.NoError(t, DB.Create(plan).Error)
	return plan
}

func insertSubscriptionOrderForPaymentGuardTest(t *testing.T, tradeNo string, userID int, planID int, paymentProvider string) {
	t.Helper()
	order := &SubscriptionOrder{
		UserId:          userID,
		PlanId:          planID,
		Money:           9.99,
		TradeNo:         tradeNo,
		PaymentMethod:   paymentProvider,
		PaymentProvider: paymentProvider,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, order.Insert())
}

func insertTopUpForPaymentGuardTest(t *testing.T, tradeNo string, userID int, paymentProvider string) {
	t.Helper()
	topUp := &TopUp{
		UserId:          userID,
		Amount:          2,
		Money:           9.99,
		TradeNo:         tradeNo,
		PaymentMethod:   paymentProvider,
		PaymentProvider: paymentProvider,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())
}

func getTopUpStatusForPaymentGuardTest(t *testing.T, tradeNo string) string {
	t.Helper()
	topUp := GetTopUpByTradeNo(tradeNo)
	require.NotNil(t, topUp)
	return topUp.Status
}

func countUserSubscriptionsForPaymentGuardTest(t *testing.T, userID int) int64 {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", userID).Count(&count).Error)
	return count
}

func getUserQuotaForPaymentGuardTest(t *testing.T, userID int) int {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("quota").Where("id = ?", userID).First(&user).Error)
	return user.Quota
}

func TestRechargeWaffoPancake_RejectsMismatchedPaymentMethod(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 101, 0)
	insertTopUpForPaymentGuardTest(t, "waffo-pancake-guard", 101, PaymentProviderStripe)

	err := RechargeWaffoPancake("waffo-pancake-guard")
	require.Error(t, err)

	topUp := GetTopUpByTradeNo("waffo-pancake-guard")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 101))
}

func TestUpdatePendingTopUpStatus_RejectsMismatchedPaymentProvider(t *testing.T) {
	testCases := []struct {
		name                    string
		tradeNo                 string
		storedPaymentProvider   string
		expectedPaymentProvider string
		targetStatus            string
	}{
		{
			name:                    "stripe expire",
			tradeNo:                 "stripe-expire-guard",
			storedPaymentProvider:   PaymentProviderCreem,
			expectedPaymentProvider: PaymentProviderStripe,
			targetStatus:            common.TopUpStatusExpired,
		},
		{
			name:                    "waffo failed",
			tradeNo:                 "waffo-failed-guard",
			storedPaymentProvider:   PaymentProviderStripe,
			expectedPaymentProvider: PaymentProviderWaffo,
			targetStatus:            common.TopUpStatusFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			truncateTables(t)
			insertUserForPaymentGuardTest(t, 150, 0)
			insertTopUpForPaymentGuardTest(t, tc.tradeNo, 150, tc.storedPaymentProvider)

			err := UpdatePendingTopUpStatus(tc.tradeNo, tc.expectedPaymentProvider, tc.targetStatus)
			require.ErrorIs(t, err, ErrPaymentMethodMismatch)
			assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, tc.tradeNo))
		})
	}
}

func TestCompleteGoPayTopUpRejectsCrossGatewayProvider(t *testing.T) {
	err := CompleteGoPayTopUp(
		"cross-gateway",
		PaymentProviderStripe,
		PaymentMethodGoPayAlipay,
		100,
		"127.0.0.1",
	)
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)
}

func TestCompleteGoPayTopUpSettlesOnceAndValidatesAmount(t *testing.T) {
	truncateTables(t)
	insertUserForPaymentGuardTest(t, 151, 0)
	topUp := &TopUp{
		UserId:          151,
		Amount:          2,
		Money:           14.60,
		TradeNo:         "gopay-settlement",
		PaymentMethod:   PaymentMethodGoPayAlipay,
		PaymentProvider: PaymentProviderGoPayAlipay,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())

	err := CompleteGoPayTopUp(
		topUp.TradeNo,
		PaymentProviderGoPayAlipay,
		PaymentMethodGoPayAlipay,
		1459,
		"127.0.0.1",
	)
	require.ErrorContains(t, err, "payment amount mismatch")
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, topUp.TradeNo))
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 151))

	require.NoError(t, CompleteGoPayTopUp(
		topUp.TradeNo,
		PaymentProviderGoPayAlipay,
		PaymentMethodGoPayAlipay,
		1460,
		"127.0.0.1",
	))
	expectedQuota := common.QuotaFromDecimal(decimal.NewFromInt(2).Mul(decimal.NewFromFloat(common.QuotaPerUnit)))
	assert.Equal(t, common.TopUpStatusSuccess, getTopUpStatusForPaymentGuardTest(t, topUp.TradeNo))
	assert.Equal(t, expectedQuota, getUserQuotaForPaymentGuardTest(t, 151))

	require.NoError(t, CompleteGoPayTopUp(
		topUp.TradeNo,
		PaymentProviderGoPayAlipay,
		PaymentMethodGoPayAlipay,
		1460,
		"127.0.0.1",
	))
	assert.Equal(t, expectedQuota, getUserQuotaForPaymentGuardTest(t, 151))
}

func TestRejectCorporateTopUpStoresReasonWithoutCreditingQuota(t *testing.T) {
	truncateTables(t)
	insertUserForPaymentGuardTest(t, 175, 100)
	insertTopUpForPaymentGuardTest(t, "corporate-reject", 175, PaymentProviderCorporate)

	require.NoError(t, RejectCorporateTopUp("corporate-reject", "  凭证金额与订单不一致  ", "127.0.0.1"))

	topUp := GetTopUpByTradeNo("corporate-reject")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusFailed, topUp.Status)
	assert.Equal(t, "凭证金额与订单不一致", topUp.RejectReason)
	assert.NotZero(t, topUp.CompleteTime)
	assert.Equal(t, 100, getUserQuotaForPaymentGuardTest(t, 175))
}

func TestRejectCorporateTopUpRequiresPendingCorporateOrderAndReason(t *testing.T) {
	testCases := []struct {
		name     string
		provider string
		reason   string
	}{
		{name: "requires corporate provider", provider: PaymentProviderStripe, reason: "invalid"},
		{name: "requires reason", provider: PaymentProviderCorporate, reason: "  "},
	}

	for index, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			truncateTables(t)
			insertUserForPaymentGuardTest(t, 180+index, 50)
			tradeNo := fmt.Sprintf("corporate-reject-guard-%d", index)
			insertTopUpForPaymentGuardTest(t, tradeNo, 180+index, test.provider)

			require.Error(t, RejectCorporateTopUp(tradeNo, test.reason, "127.0.0.1"))
			assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, tradeNo))
			assert.Equal(t, 50, getUserQuotaForPaymentGuardTest(t, 180+index))
		})
	}
}

func TestCancelCorporateTopUpOnlyAllowsOwnerPendingCorporateOrder(t *testing.T) {
	truncateTables(t)
	insertUserForPaymentGuardTest(t, 190, 75)
	insertTopUpForPaymentGuardTest(t, "corporate-cancel", 190, PaymentProviderCorporate)

	require.Error(t, CancelCorporateTopUp("corporate-cancel", 191, "127.0.0.1"))
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, "corporate-cancel"))

	require.NoError(t, CancelCorporateTopUp("corporate-cancel", 190, "127.0.0.1"))
	topUp := GetTopUpByTradeNo("corporate-cancel")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusCanceled, topUp.Status)
	assert.NotZero(t, topUp.CompleteTime)
	assert.Equal(t, 75, getUserQuotaForPaymentGuardTest(t, 190))

	require.Error(t, CancelCorporateTopUp("corporate-cancel", 190, "127.0.0.1"))
}

func TestCompleteSubscriptionOrder_RejectsMismatchedPaymentProvider(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 202, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 301)
	insertSubscriptionOrderForPaymentGuardTest(t, "sub-guard-order", 202, plan.Id, PaymentProviderStripe)

	err := CompleteSubscriptionOrder("sub-guard-order", `{"provider":"epay"}`, PaymentProviderEpay, "alipay")
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)

	order := GetSubscriptionOrderByTradeNo("sub-guard-order")
	require.NotNil(t, order)
	assert.Equal(t, common.TopUpStatusPending, order.Status)
	assert.Zero(t, countUserSubscriptionsForPaymentGuardTest(t, 202))

	topUp := GetTopUpByTradeNo("sub-guard-order")
	assert.Nil(t, topUp)
}

func TestExpireSubscriptionOrder_RejectsMismatchedPaymentProvider(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 303, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 401)
	insertSubscriptionOrderForPaymentGuardTest(t, "sub-expire-guard", 303, plan.Id, PaymentProviderStripe)

	err := ExpireSubscriptionOrder("sub-expire-guard", PaymentProviderCreem)
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)

	order := GetSubscriptionOrderByTradeNo("sub-expire-guard")
	require.NotNil(t, order)
	assert.Equal(t, common.TopUpStatusPending, order.Status)
}
