//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

type paymentFulfillmentGroupRepoStub struct {
	groupRepoNoop
	groups map[int64]*Group
}

func (s *paymentFulfillmentGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	group := s.groups[id]
	if group == nil {
		return nil, ErrGroupNotFound
	}
	cp := *group
	return &cp, nil
}

func TestNormalizePlanSubscriptionGroupIDsIncludesPrimaryFirst(t *testing.T) {
	t.Parallel()

	got := normalizePlanSubscriptionGroupIDs(10, []int64{24, 10, 0, -1, 24, 31})

	require.Equal(t, []int64{10, 24, 31}, got)
}

func TestSubscriptionFulfillmentAssignsOnlyPrimaryOrderGroup(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	user, err := client.User.Create().
		SetEmail("kiro-bundle@example.com").
		SetPasswordHash("hash").
		SetUsername("kiro-bundle").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(128).
		SetPayAmount(128).
		SetRechargeCode("PAY-MULTI-GROUP").
		SetPaymentType("alipay").
		SetPaymentTradeNo("trade-multi-group").
		SetOrderType(payment.OrderTypeSubscription).
		SetStatus(OrderStatusRecharging).
		SetExpiresAt(time.Now().Add(15 * time.Minute)).
		SetClientIP("127.0.0.1").
		SetSrcHost("example.test").
		SetPlanID(7).
		SetSubscriptionGroupID(10).
		SetSubscriptionGroupIds([]int64{10, 24}).
		SetSubscriptionDays(30).
		Save(ctx)
	require.NoError(t, err)

	groupRepo := &paymentFulfillmentGroupRepoStub{
		groups: map[int64]*Group{
			10: {ID: 10, Name: "Codex OpenAI", Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription},
			24: {ID: 24, Name: "Kiro", Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription},
		},
	}
	subRepo := newSubscriptionUserSubRepoStub()
	subscriptionSvc := NewSubscriptionService(groupRepo, subRepo, nil, client, nil)
	svc := &PaymentService{entClient: client, groupRepo: groupRepo, subscriptionSvc: subscriptionSvc}

	err = svc.doSub(ctx, order)
	require.NoError(t, err)

	sub, err := subRepo.GetByUserIDAndGroupID(ctx, user.ID, 10)
	require.NoError(t, err)
	require.Equal(t, SubscriptionStatusActive, sub.Status)
	require.Contains(t, sub.Notes, "payment order")

	_, err = subRepo.GetByUserIDAndGroupID(ctx, user.ID, 24)
	require.ErrorIs(t, err, ErrSubscriptionNotFound)

	auditCount, err := client.PaymentAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, auditCount, 1)

	reloaded, err := client.PaymentOrder.Query().Where(paymentorder.IDEQ(order.ID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, reloaded.Status)
}

func TestSubscriptionFulfillmentExtendsExistingPrimaryOrderGroupOnly(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	user, err := client.User.Create().
		SetEmail("kiro-bundle-retry@example.com").
		SetPasswordHash("hash").
		SetUsername("kiro-bundle-retry").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(128).
		SetPayAmount(128).
		SetRechargeCode("PAY-MULTI-GROUP-RETRY").
		SetPaymentType("alipay").
		SetPaymentTradeNo("trade-multi-group-retry").
		SetOrderType(payment.OrderTypeSubscription).
		SetStatus(OrderStatusRecharging).
		SetExpiresAt(time.Now().Add(15 * time.Minute)).
		SetClientIP("127.0.0.1").
		SetSrcHost("example.test").
		SetPlanID(7).
		SetSubscriptionGroupID(10).
		SetSubscriptionGroupIds([]int64{10, 24}).
		SetSubscriptionDays(30).
		Save(ctx)
	require.NoError(t, err)

	groupRepo := &paymentFulfillmentGroupRepoStub{
		groups: map[int64]*Group{
			10: {ID: 10, Name: "Codex OpenAI", Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription},
			24: {ID: 24, Name: "Kiro", Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription},
		},
	}
	subRepo := newSubscriptionUserSubRepoStub()
	existingExpiry := time.Now().Add(30 * 24 * time.Hour)
	subRepo.seed(&UserSubscription{
		ID:        100,
		UserID:    user.ID,
		GroupID:   10,
		Status:    SubscriptionStatusActive,
		ExpiresAt: existingExpiry,
		Notes:     "payment order prior run",
	})
	subscriptionSvc := NewSubscriptionService(groupRepo, subRepo, nil, client, nil)
	svc := &PaymentService{entClient: client, groupRepo: groupRepo, subscriptionSvc: subscriptionSvc}

	err = svc.doSub(ctx, order)
	require.NoError(t, err)

	sub, err := subRepo.GetByUserIDAndGroupID(ctx, user.ID, 10)
	require.NoError(t, err)
	require.Greater(t, sub.ExpiresAt.Sub(existingExpiry), 29*24*time.Hour)
	_, err = subRepo.GetByUserIDAndGroupID(ctx, user.ID, 24)
	require.ErrorIs(t, err, ErrSubscriptionNotFound)
	require.Equal(t, 0, subRepo.createCalls, "renewal should extend the existing primary subscription instead of creating an add-on subscription")
}
