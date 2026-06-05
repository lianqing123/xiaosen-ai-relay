//go:build unit

package service

import (
	"testing"
)

// TestBuildUsageBillingCommand_SubscriptionAppliesRateMultiplier locks in the fix
// that subscription-mode billing honours the group (and any user-specific) rate
// multiplier — i.e. cmd.SubscriptionCost tracks ActualCost (= TotalCost *
// RateMultiplier), not raw TotalCost.
func TestBuildUsageBillingCommand_SubscriptionAppliesRateMultiplier(t *testing.T) {
	t.Parallel()

	groupID := int64(7)
	subID := int64(42)

	tests := []struct {
		name           string
		totalCost      float64
		actualCost     float64
		isSubscription bool
		wantSub        float64
		wantBalance    float64
	}{
		{
			name:           "subscription with 2x multiplier consumes 2x quota",
			totalCost:      1.0,
			actualCost:     2.0,
			isSubscription: true,
			wantSub:        2.0,
			wantBalance:    0,
		},
		{
			name:           "subscription with 0.5x multiplier consumes 0.5x quota",
			totalCost:      1.0,
			actualCost:     0.5,
			isSubscription: true,
			wantSub:        0.5,
			wantBalance:    0,
		},
		{
			name:           "free subscription (multiplier 0) consumes no quota",
			totalCost:      1.0,
			actualCost:     0,
			isSubscription: true,
			wantSub:        0,
			wantBalance:    0,
		},
		{
			name:           "balance billing keeps using ActualCost (regression)",
			totalCost:      1.0,
			actualCost:     2.0,
			isSubscription: false,
			wantSub:        0,
			wantBalance:    2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &postUsageBillingParams{
				Cost:               &CostBreakdown{TotalCost: tt.totalCost, ActualCost: tt.actualCost},
				User:               &User{ID: 1},
				APIKey:             &APIKey{ID: 2, GroupID: &groupID},
				Account:            &Account{ID: 3},
				Subscription:       &UserSubscription{ID: subID},
				IsSubscriptionBill: tt.isSubscription,
			}

			cmd := buildUsageBillingCommand("req-1", nil, p)
			if cmd == nil {
				t.Fatal("buildUsageBillingCommand returned nil")
			}
			if cmd.SubscriptionCost != tt.wantSub {
				t.Errorf("SubscriptionCost = %v, want %v", cmd.SubscriptionCost, tt.wantSub)
			}
			if cmd.BalanceCost != tt.wantBalance {
				t.Errorf("BalanceCost = %v, want %v", cmd.BalanceCost, tt.wantBalance)
			}
		})
	}
}

func TestFinalizePostUsageBillingUpdatesResolvedSubscriptionGroupCache(t *testing.T) {
	t.Parallel()

	routeGroupID := int64(24)
	primaryGroupID := int64(13)
	cache := &billingCacheWorkerStub{}

	finalizePostUsageBilling(&postUsageBillingParams{
		Cost: &CostBreakdown{ActualCost: 0.42},
		User: &User{ID: 200},
		APIKey: &APIKey{
			ID:      300,
			GroupID: &routeGroupID,
		},
		Account: &Account{ID: 400},
		Subscription: &UserSubscription{
			ID:      500,
			UserID:  200,
			GroupID: primaryGroupID,
		},
		IsSubscriptionBill: true,
	}, &billingDeps{
		billingCacheService: &BillingCacheService{cache: cache},
		deferredService:     &DeferredService{},
	}, &UsageBillingApplyResult{Applied: true})

	if cache.lastSubscriptionGroupID != primaryGroupID {
		t.Fatalf("subscription cache group = %d, want primary package group %d", cache.lastSubscriptionGroupID, primaryGroupID)
	}
	if cache.lastSubscriptionUserID != 200 {
		t.Fatalf("subscription cache user = %d, want 200", cache.lastSubscriptionUserID)
	}
	if cache.lastSubscriptionCost != 0.42 {
		t.Fatalf("subscription cache cost = %v, want 0.42", cache.lastSubscriptionCost)
	}
}
