package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type billingCacheWorkerStub struct {
	balanceUpdates          int64
	subscriptionUpdates     int64
	lastSubscriptionUserID  int64
	lastSubscriptionGroupID int64
	lastSubscriptionCost    float64
	balance                 float64
	balanceErr              error
	subscription            *SubscriptionCacheData
	subscriptionErr         error
}

func (b *billingCacheWorkerStub) GetUserBalance(ctx context.Context, userID int64) (float64, error) {
	if b.balanceErr != nil {
		return 0, b.balanceErr
	}
	if b.balance != 0 {
		return b.balance, nil
	}
	return 0, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetUserBalance(ctx context.Context, userID int64, balance float64) error {
	atomic.AddInt64(&b.balanceUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) DeductUserBalance(ctx context.Context, userID int64, amount float64) error {
	atomic.AddInt64(&b.balanceUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) InvalidateUserBalance(ctx context.Context, userID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetSubscriptionCache(ctx context.Context, userID, groupID int64) (*SubscriptionCacheData, error) {
	if b.subscriptionErr != nil {
		return nil, b.subscriptionErr
	}
	if b.subscription != nil {
		return b.subscription, nil
	}
	return nil, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetSubscriptionCache(ctx context.Context, userID, groupID int64, data *SubscriptionCacheData) error {
	atomic.AddInt64(&b.subscriptionUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) UpdateSubscriptionUsage(ctx context.Context, userID, groupID int64, cost float64) error {
	atomic.AddInt64(&b.subscriptionUpdates, 1)
	b.lastSubscriptionUserID = userID
	b.lastSubscriptionGroupID = groupID
	b.lastSubscriptionCost = cost
	return nil
}

func (b *billingCacheWorkerStub) InvalidateSubscriptionCache(ctx context.Context, userID, groupID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetAPIKeyRateLimit(ctx context.Context, keyID int64) (*APIKeyRateLimitCacheData, error) {
	return nil, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetAPIKeyRateLimit(ctx context.Context, keyID int64, data *APIKeyRateLimitCacheData) error {
	return nil
}

func (b *billingCacheWorkerStub) UpdateAPIKeyRateLimitUsage(ctx context.Context, keyID int64, cost float64) error {
	return nil
}

func (b *billingCacheWorkerStub) InvalidateAPIKeyRateLimit(ctx context.Context, keyID int64) error {
	return nil
}

func TestBillingCacheServiceQueueHighLoad(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	start := time.Now()
	for i := 0; i < cacheWriteBufferSize*2; i++ {
		svc.QueueDeductBalance(1, 1)
	}
	require.Less(t, time.Since(start), 2*time.Second)

	svc.QueueUpdateSubscriptionUsage(1, 2, 1.5)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.balanceUpdates) > 0
	}, 2*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.subscriptionUpdates) > 0
	}, 2*time.Second, 10*time.Millisecond)
}

func TestBillingCacheServiceEnqueueAfterStopReturnsFalse(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{})
	svc.Stop()

	enqueued := svc.enqueueCacheWrite(cacheWriteTask{
		kind:   cacheWriteDeductBalance,
		userID: 1,
		amount: 1,
	})
	require.False(t, enqueued)
}

func TestCheckBillingEligibilityAllowsBalanceFallbackWhenSubscriptionQuotaExceeded(t *testing.T) {
	limit := 10.0
	cache := &billingCacheWorkerStub{
		balance: 3,
		subscription: &SubscriptionCacheData{
			Status:       SubscriptionStatusActive,
			ExpiresAt:    time.Now().Add(time.Hour),
			DailyUsage:   limit,
			WeeklyUsage:  1,
			MonthlyUsage: 1,
		},
	}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(
		context.Background(),
		&User{ID: 1, SubscriptionOveragePaygEnabled: true},
		&APIKey{ID: 2},
		&Group{
			ID:               3,
			SubscriptionType: SubscriptionTypeSubscription,
			DailyLimitUSD:    &limit,
		},
		&UserSubscription{ID: 4},
	)

	require.NoError(t, err)
}

func TestCheckBillingEligibilityRejectsClaudeBalanceFallbackWhenOverageSwitchDisabled(t *testing.T) {
	limit := 10.0
	cache := &billingCacheWorkerStub{
		balance: 3,
		subscription: &SubscriptionCacheData{
			Status:       SubscriptionStatusActive,
			ExpiresAt:    time.Now().Add(time.Hour),
			DailyUsage:   limit,
			WeeklyUsage:  1,
			MonthlyUsage: 1,
		},
	}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(
		context.Background(),
		&User{ID: 1, Balance: 3},
		&APIKey{ID: 2},
		&Group{
			ID:                   24,
			Name:                 "Claude",
			Platform:             PlatformAnthropic,
			SubscriptionType:     SubscriptionTypeSubscription,
			DailyLimitUSD:        &limit,
			SupportedModelScopes: []string{"claude"},
		},
		&UserSubscription{ID: 4},
	)

	require.ErrorIs(t, err, ErrDailyLimitExceeded)
}

func TestShouldBillSubscriptionKeepsSubscriptionForClaudeWhenOverageSwitchDisabled(t *testing.T) {
	limit := 10.0
	cache := &billingCacheWorkerStub{
		balance: 3,
		subscription: &SubscriptionCacheData{
			Status:       SubscriptionStatusActive,
			ExpiresAt:    time.Now().Add(time.Hour),
			DailyUsage:   limit,
			WeeklyUsage:  1,
			MonthlyUsage: 1,
		},
	}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	got := svc.ShouldBillSubscription(
		context.Background(),
		&User{ID: 1, Balance: 3},
		&Group{
			ID:                   24,
			Name:                 "Claude",
			Platform:             PlatformAnthropic,
			SubscriptionType:     SubscriptionTypeSubscription,
			DailyLimitUSD:        &limit,
			SupportedModelScopes: []string{"claude"},
		},
		&UserSubscription{ID: 4},
	)

	require.True(t, got)
}

func TestShouldBillSubscriptionUsesBalanceForClaudeWhenOverageSwitchEnabled(t *testing.T) {
	limit := 10.0
	cache := &billingCacheWorkerStub{
		balance: 3,
		subscription: &SubscriptionCacheData{
			Status:       SubscriptionStatusActive,
			ExpiresAt:    time.Now().Add(time.Hour),
			DailyUsage:   limit,
			WeeklyUsage:  1,
			MonthlyUsage: 1,
		},
	}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	got := svc.ShouldBillSubscription(
		context.Background(),
		&User{ID: 1, Balance: 3, SubscriptionOveragePaygEnabled: true},
		&Group{
			ID:                   24,
			Name:                 "Claude",
			Platform:             PlatformAnthropic,
			SubscriptionType:     SubscriptionTypeSubscription,
			DailyLimitUSD:        &limit,
			SupportedModelScopes: []string{"claude"},
		},
		&UserSubscription{ID: 4},
	)

	require.False(t, got)
}

func TestCheckBillingEligibilityIgnoresNegativeBalanceForActiveSubscriptionWithinQuota(t *testing.T) {
	limit := 10.0
	cache := &billingCacheWorkerStub{
		balance: -0.03,
		subscription: &SubscriptionCacheData{
			Status:       SubscriptionStatusActive,
			ExpiresAt:    time.Now().Add(time.Hour),
			DailyUsage:   1,
			WeeklyUsage:  1,
			MonthlyUsage: 1,
		},
	}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(
		context.Background(),
		&User{ID: 1},
		&APIKey{ID: 2},
		&Group{
			ID:               3,
			SubscriptionType: SubscriptionTypeSubscription,
			DailyLimitUSD:    &limit,
		},
		&UserSubscription{ID: 4},
	)

	require.NoError(t, err)
}

func TestCheckBillingEligibilityRejectsSubscriptionQuotaExceededWhenFallbackDisabled(t *testing.T) {
	limit := 10.0
	cache := &billingCacheWorkerStub{
		balance: 3,
		subscription: &SubscriptionCacheData{
			Status:      SubscriptionStatusActive,
			ExpiresAt:   time.Now().Add(time.Hour),
			DailyUsage:  limit,
			WeeklyUsage: 1,
		},
	}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	err := svc.CheckBillingEligibility(
		context.Background(),
		&User{ID: 1},
		&APIKey{ID: 2},
		&Group{
			ID:               3,
			SubscriptionType: SubscriptionTypeSubscription,
			DailyLimitUSD:    &limit,
		},
		&UserSubscription{ID: 4},
	)

	require.ErrorIs(t, err, ErrDailyLimitExceeded)
}
