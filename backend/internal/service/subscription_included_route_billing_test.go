//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type includedRouteBillingSubRepoStub struct {
	userSubRepoNoop
	subs []UserSubscription
}

func (s includedRouteBillingSubRepoStub) ListActiveByUserID(_ context.Context, userID int64) ([]UserSubscription, error) {
	out := make([]UserSubscription, 0, len(s.subs))
	for _, sub := range s.subs {
		if sub.UserID != userID {
			continue
		}
		cp := sub
		out = append(out, cp)
	}
	return out, nil
}

func TestResolveIncludedRouteBillingSubscriptionUsesPrimaryPlanSubscription(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	user, err := client.User.Create().
		SetEmail("kiro-quota@example.com").
		SetPasswordHash("hash").
		SetUsername("kiro-quota").
		Save(ctx)
	require.NoError(t, err)

	primaryGroup, err := client.Group.Create().
		SetName("Codex Pro").
		SetPlatform(PlatformOpenAI).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		SetMonthlyLimitUsd(1000).
		Save(ctx)
	require.NoError(t, err)

	kiroGroup, err := client.Group.Create().
		SetName("Kiro").
		SetPlatform(PlatformAnthropic).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SubscriptionPlan.Create().
		SetName("Codex Pro Bundle").
		SetGroupID(primaryGroup.ID).
		SetIncludedGroupIds([]int64{kiroGroup.ID}).
		SetPrice(399).
		SetValidityDays(30).
		Save(ctx)
	require.NoError(t, err)

	now := time.Now()
	primarySub, err := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(primaryGroup.ID).
		SetStartsAt(now.Add(-time.Hour)).
		SetExpiresAt(now.Add(30 * 24 * time.Hour)).
		SetStatus(SubscriptionStatusActive).
		Save(ctx)
	require.NoError(t, err)

	kiroSub, err := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(kiroGroup.ID).
		SetStartsAt(now.Add(-time.Hour)).
		SetExpiresAt(now.Add(30 * 24 * time.Hour)).
		SetStatus(SubscriptionStatusActive).
		Save(ctx)
	require.NoError(t, err)

	subRepo := includedRouteBillingSubRepoStub{subs: []UserSubscription{
		{
			ID:        primarySub.ID,
			UserID:    user.ID,
			GroupID:   primaryGroup.ID,
			ExpiresAt: now.Add(30 * 24 * time.Hour),
			Status:    SubscriptionStatusActive,
			Group: &Group{
				ID:               primaryGroup.ID,
				SubscriptionType: SubscriptionTypeSubscription,
				Status:           StatusActive,
				MonthlyLimitUSD:  float64Ptr(1000),
			},
		},
		{
			ID:        kiroSub.ID,
			UserID:    user.ID,
			GroupID:   kiroGroup.ID,
			ExpiresAt: now.Add(30 * 24 * time.Hour),
			Status:    SubscriptionStatusActive,
			Group: &Group{
				ID:               kiroGroup.ID,
				SubscriptionType: SubscriptionTypeSubscription,
				Status:           StatusActive,
			},
		},
	}}
	svc := NewSubscriptionService(groupRepoNoop{}, subRepo, nil, client, nil)

	resolvedSub, resolvedGroup, err := svc.ResolveIncludedRouteBillingSubscription(ctx, user.ID, &Group{
		ID:               kiroGroup.ID,
		SubscriptionType: SubscriptionTypeSubscription,
		Status:           StatusActive,
	}, &UserSubscription{
		ID:      kiroSub.ID,
		UserID:  user.ID,
		GroupID: kiroGroup.ID,
		Group: &Group{
			ID:               kiroGroup.ID,
			SubscriptionType: SubscriptionTypeSubscription,
			Status:           StatusActive,
		},
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, resolvedSub)
	require.Equal(t, primarySub.ID, resolvedSub.ID)
	require.Equal(t, primaryGroup.ID, resolvedSub.GroupID)
	require.NotNil(t, resolvedGroup)
	require.Equal(t, primaryGroup.ID, resolvedGroup.ID)
	require.NotNil(t, resolvedGroup.MonthlyLimitUSD)
	require.Equal(t, 1000.0, *resolvedGroup.MonthlyLimitUSD)
}

func TestResolveIncludedRouteBillingSubscriptionUsesPrimaryPlanWithoutRouteSubscription(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	user, err := client.User.Create().
		SetEmail("claude-primary-only@example.com").
		SetPasswordHash("hash").
		SetUsername("claude-primary-only").
		Save(ctx)
	require.NoError(t, err)

	primaryGroup, err := client.Group.Create().
		SetName("Codex Plus").
		SetPlatform(PlatformOpenAI).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		SetDailyLimitUsd(20).
		SetMonthlyLimitUsd(500).
		Save(ctx)
	require.NoError(t, err)

	claudeGroup, err := client.Group.Create().
		SetName("Claude").
		SetPlatform(PlatformAnthropic).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SubscriptionPlan.Create().
		SetName("Codex Plus Bundle").
		SetGroupID(primaryGroup.ID).
		SetIncludedGroupIds([]int64{claudeGroup.ID}).
		SetPrice(399).
		SetValidityDays(30).
		Save(ctx)
	require.NoError(t, err)

	now := time.Now()
	primarySub, err := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(primaryGroup.ID).
		SetStartsAt(now.Add(-time.Hour)).
		SetExpiresAt(now.Add(30 * 24 * time.Hour)).
		SetStatus(SubscriptionStatusActive).
		Save(ctx)
	require.NoError(t, err)

	subRepo := includedRouteBillingSubRepoStub{subs: []UserSubscription{
		{
			ID:        primarySub.ID,
			UserID:    user.ID,
			GroupID:   primaryGroup.ID,
			ExpiresAt: now.Add(30 * 24 * time.Hour),
			Status:    SubscriptionStatusActive,
			Group: &Group{
				ID:               primaryGroup.ID,
				Name:             "Codex Plus",
				SubscriptionType: SubscriptionTypeSubscription,
				Status:           StatusActive,
				DailyLimitUSD:    float64Ptr(20),
				MonthlyLimitUSD:  float64Ptr(500),
			},
		},
	}}
	svc := NewSubscriptionService(groupRepoNoop{}, subRepo, nil, client, nil)

	resolvedSub, resolvedGroup, err := svc.ResolveIncludedRouteBillingSubscription(ctx, user.ID, &Group{
		ID:               claudeGroup.ID,
		Name:             "Claude",
		SubscriptionType: SubscriptionTypeSubscription,
		Status:           StatusActive,
	}, nil, nil)

	require.NoError(t, err)
	require.NotNil(t, resolvedSub)
	require.Equal(t, primarySub.ID, resolvedSub.ID)
	require.Equal(t, primaryGroup.ID, resolvedSub.GroupID)
	require.NotNil(t, resolvedGroup)
	require.Equal(t, primaryGroup.ID, resolvedGroup.ID)
	require.NotNil(t, resolvedGroup.DailyLimitUSD)
	require.Equal(t, 20.0, *resolvedGroup.DailyLimitUSD)
}

func TestResolveIncludedRouteBillingSubscriptionUsesPreferredPrimaryPackage(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	user, err := client.User.Create().
		SetEmail("claude-preferred-package@example.com").
		SetPasswordHash("hash").
		SetUsername("claude-preferred-package").
		Save(ctx)
	require.NoError(t, err)

	starterGroup, err := client.Group.Create().
		SetName("Codex Starter").
		SetPlatform(PlatformOpenAI).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		SetMonthlyLimitUsd(100).
		Save(ctx)
	require.NoError(t, err)

	proGroup, err := client.Group.Create().
		SetName("Codex Pro").
		SetPlatform(PlatformOpenAI).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		SetMonthlyLimitUsd(1000).
		Save(ctx)
	require.NoError(t, err)

	claudeGroup, err := client.Group.Create().
		SetName("Claude").
		SetPlatform(PlatformAnthropic).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SubscriptionPlan.Create().
		SetName("Codex Starter Bundle").
		SetGroupID(starterGroup.ID).
		SetIncludedGroupIds([]int64{claudeGroup.ID}).
		SetPrice(99).
		SetValidityDays(30).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.SubscriptionPlan.Create().
		SetName("Codex Pro Bundle").
		SetGroupID(proGroup.ID).
		SetIncludedGroupIds([]int64{claudeGroup.ID}).
		SetPrice(399).
		SetValidityDays(30).
		Save(ctx)
	require.NoError(t, err)

	now := time.Now()
	subRepo := includedRouteBillingSubRepoStub{subs: []UserSubscription{
		{
			ID:        1001,
			UserID:    user.ID,
			GroupID:   starterGroup.ID,
			ExpiresAt: now.Add(30 * 24 * time.Hour),
			Status:    SubscriptionStatusActive,
			Group: &Group{
				ID:               starterGroup.ID,
				Name:             "Codex Starter",
				SubscriptionType: SubscriptionTypeSubscription,
				Status:           StatusActive,
				MonthlyLimitUSD:  float64Ptr(100),
			},
		},
		{
			ID:        1002,
			UserID:    user.ID,
			GroupID:   proGroup.ID,
			ExpiresAt: now.Add(30 * 24 * time.Hour),
			Status:    SubscriptionStatusActive,
			Group: &Group{
				ID:               proGroup.ID,
				Name:             "Codex Pro",
				SubscriptionType: SubscriptionTypeSubscription,
				Status:           StatusActive,
				MonthlyLimitUSD:  float64Ptr(1000),
			},
		},
	}}
	svc := NewSubscriptionService(groupRepoNoop{}, subRepo, nil, client, nil)

	preferredGroupID := starterGroup.ID
	resolvedSub, resolvedGroup, err := svc.ResolveIncludedRouteBillingSubscription(ctx, user.ID, &Group{
		ID:               claudeGroup.ID,
		Name:             "Claude",
		SubscriptionType: SubscriptionTypeSubscription,
		Status:           StatusActive,
	}, nil, &preferredGroupID)

	require.NoError(t, err)
	require.NotNil(t, resolvedSub)
	require.Equal(t, int64(1001), resolvedSub.ID)
	require.Equal(t, starterGroup.ID, resolvedSub.GroupID)
	require.NotNil(t, resolvedGroup)
	require.Equal(t, starterGroup.ID, resolvedGroup.ID)
}

func TestResolveIncludedRouteBillingSubscriptionIgnoresStaleDirectClaudePreference(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	user, err := client.User.Create().
		SetEmail("claude-stale-direct-preference@example.com").
		SetPasswordHash("hash").
		SetUsername("claude-stale-direct-preference").
		Save(ctx)
	require.NoError(t, err)

	primaryGroup, err := client.Group.Create().
		SetName("Codex Pro").
		SetPlatform(PlatformOpenAI).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		SetMonthlyLimitUsd(1000).
		Save(ctx)
	require.NoError(t, err)

	claudeGroup, err := client.Group.Create().
		SetName("Claude").
		SetPlatform(PlatformAnthropic).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SubscriptionPlan.Create().
		SetName("Codex Pro Bundle").
		SetGroupID(primaryGroup.ID).
		SetIncludedGroupIds([]int64{claudeGroup.ID}).
		SetPrice(399).
		SetValidityDays(30).
		Save(ctx)
	require.NoError(t, err)

	now := time.Now()
	subRepo := includedRouteBillingSubRepoStub{subs: []UserSubscription{
		{
			ID:        1002,
			UserID:    user.ID,
			GroupID:   primaryGroup.ID,
			ExpiresAt: now.Add(30 * 24 * time.Hour),
			Status:    SubscriptionStatusActive,
			Group: &Group{
				ID:               primaryGroup.ID,
				Name:             "Codex Pro",
				SubscriptionType: SubscriptionTypeSubscription,
				Status:           StatusActive,
				MonthlyLimitUSD:  float64Ptr(1000),
			},
		},
	}}
	svc := NewSubscriptionService(groupRepoNoop{}, subRepo, nil, client, nil)

	stalePreferredGroupID := claudeGroup.ID
	resolvedSub, resolvedGroup, err := svc.ResolveIncludedRouteBillingSubscription(ctx, user.ID, &Group{
		ID:               claudeGroup.ID,
		Name:             "Claude",
		Platform:         PlatformAnthropic,
		SubscriptionType: SubscriptionTypeSubscription,
		Status:           StatusActive,
	}, nil, &stalePreferredGroupID)

	require.NoError(t, err)
	require.NotNil(t, resolvedSub)
	require.Equal(t, int64(1002), resolvedSub.ID)
	require.Equal(t, primaryGroup.ID, resolvedSub.GroupID)
	require.NotNil(t, resolvedGroup)
	require.Equal(t, primaryGroup.ID, resolvedGroup.ID)
}

func TestResolveIncludedRouteBillingSubscriptionRejectsDirectClaudeRouteSubscriptionWhenNoPrimaryPackage(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	user, err := client.User.Create().
		SetEmail("claude-direct-only@example.com").
		SetPasswordHash("hash").
		SetUsername("claude-direct-only").
		Save(ctx)
	require.NoError(t, err)

	primaryGroup, err := client.Group.Create().
		SetName("Codex Pro Required").
		SetPlatform(PlatformOpenAI).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		SetMonthlyLimitUsd(1000).
		Save(ctx)
	require.NoError(t, err)

	claudeGroup, err := client.Group.Create().
		SetName("Claude Required").
		SetPlatform(PlatformAnthropic).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SubscriptionPlan.Create().
		SetName("Codex Pro Required Bundle").
		SetGroupID(primaryGroup.ID).
		SetIncludedGroupIds([]int64{claudeGroup.ID}).
		SetPrice(399).
		SetValidityDays(30).
		Save(ctx)
	require.NoError(t, err)

	now := time.Now()
	claudeSub, err := client.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(claudeGroup.ID).
		SetStartsAt(now.Add(-time.Hour)).
		SetExpiresAt(now.Add(30 * 24 * time.Hour)).
		SetStatus(SubscriptionStatusActive).
		Save(ctx)
	require.NoError(t, err)

	subRepo := includedRouteBillingSubRepoStub{subs: []UserSubscription{
		{
			ID:        claudeSub.ID,
			UserID:    user.ID,
			GroupID:   claudeGroup.ID,
			ExpiresAt: now.Add(30 * 24 * time.Hour),
			Status:    SubscriptionStatusActive,
			Group: &Group{
				ID:               claudeGroup.ID,
				SubscriptionType: SubscriptionTypeSubscription,
				Status:           StatusActive,
			},
		},
	}}
	svc := NewSubscriptionService(groupRepoNoop{}, subRepo, nil, client, nil)

	resolvedSub, resolvedGroup, err := svc.ResolveIncludedRouteBillingSubscription(ctx, user.ID, &Group{
		ID:               claudeGroup.ID,
		Name:             "Claude",
		Platform:         PlatformAnthropic,
		SubscriptionType: SubscriptionTypeSubscription,
		Status:           StatusActive,
	}, &UserSubscription{
		ID:      claudeSub.ID,
		UserID:  user.ID,
		GroupID: claudeGroup.ID,
		Group: &Group{
			ID:               claudeGroup.ID,
			SubscriptionType: SubscriptionTypeSubscription,
			Status:           StatusActive,
		},
	}, nil)

	require.ErrorIs(t, err, ErrSubscriptionNotFound)
	require.Nil(t, resolvedSub)
	require.Nil(t, resolvedGroup)
}
