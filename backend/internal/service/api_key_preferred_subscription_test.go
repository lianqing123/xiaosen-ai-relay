//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type preferredSubscriptionSubRepoStub struct {
	userSubRepoNoop
	subs map[int64]*UserSubscription
}

func (s preferredSubscriptionSubRepoStub) GetActiveByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*UserSubscription, error) {
	sub, ok := s.subs[groupID]
	if !ok || sub.UserID != userID {
		return nil, ErrSubscriptionNotFound
	}
	cp := *sub
	return &cp, nil
}

func TestAPIKeyPreferredSubscriptionAllowsIncludedPrimaryPackage(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	routeGroup, err := client.Group.Create().
		SetName("Claude").
		SetPlatform(PlatformAnthropic).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	primaryGroup, err := client.Group.Create().
		SetName("Codex Pro").
		SetPlatform(PlatformOpenAI).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SubscriptionPlan.Create().
		SetName("Codex Pro Bundle").
		SetGroupID(primaryGroup.ID).
		SetIncludedGroupIds([]int64{routeGroup.ID}).
		SetPrice(399).
		SetValidityDays(30).
		Save(ctx)
	require.NoError(t, err)

	subRepo := preferredSubscriptionSubRepoStub{subs: map[int64]*UserSubscription{
		primaryGroup.ID: {
			UserID:  99,
			GroupID: primaryGroup.ID,
			Group: &Group{
				ID:               primaryGroup.ID,
				SubscriptionType: SubscriptionTypeSubscription,
				Status:           StatusActive,
			},
		},
	}}
	svc := NewAPIKeyService(nil, nil, nil, subRepo, nil, nil, client, nil)

	preferredGroupID := primaryGroup.ID
	got, err := svc.normalizePreferredSubscriptionGroupID(ctx, &User{ID: 99}, &Group{
		ID:               routeGroup.ID,
		Name:             "Claude",
		Platform:         PlatformAnthropic,
		SubscriptionType: SubscriptionTypeSubscription,
		Status:           StatusActive,
	}, &preferredGroupID)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, primaryGroup.ID, *got)
}

func TestAPIKeyPreferredSubscriptionRejectsPackageThatDoesNotIncludeRoute(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	routeGroup, err := client.Group.Create().
		SetName("Claude").
		SetPlatform(PlatformAnthropic).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	unrelatedGroup, err := client.Group.Create().
		SetName("Other Subscription").
		SetPlatform(PlatformOpenAI).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SubscriptionPlan.Create().
		SetName("Other Bundle").
		SetGroupID(unrelatedGroup.ID).
		SetIncludedGroupIds([]int64{999999}).
		SetPrice(99).
		SetValidityDays(30).
		Save(ctx)
	require.NoError(t, err)

	subRepo := preferredSubscriptionSubRepoStub{subs: map[int64]*UserSubscription{
		unrelatedGroup.ID: {
			UserID:  99,
			GroupID: unrelatedGroup.ID,
			Group: &Group{
				ID:               unrelatedGroup.ID,
				SubscriptionType: SubscriptionTypeSubscription,
				Status:           StatusActive,
			},
		},
	}}
	svc := NewAPIKeyService(nil, nil, nil, subRepo, nil, nil, client, nil)

	preferredGroupID := unrelatedGroup.ID
	got, err := svc.normalizePreferredSubscriptionGroupID(ctx, &User{ID: 99}, &Group{
		ID:               routeGroup.ID,
		Name:             "Claude",
		Platform:         PlatformAnthropic,
		SubscriptionType: SubscriptionTypeSubscription,
		Status:           StatusActive,
	}, &preferredGroupID)

	require.ErrorIs(t, err, ErrPreferredSubscriptionNotAllowed)
	require.Nil(t, got)
}

func TestAPIKeyPreferredSubscriptionRejectsDirectClaudeRouteSubscription(t *testing.T) {
	ctx := context.Background()
	routeGroupID := int64(24)

	subRepo := preferredSubscriptionSubRepoStub{subs: map[int64]*UserSubscription{
		routeGroupID: {
			UserID:  99,
			GroupID: routeGroupID,
			Group: &Group{
				ID:               routeGroupID,
				SubscriptionType: SubscriptionTypeSubscription,
				Status:           StatusActive,
			},
		},
	}}
	svc := NewAPIKeyService(nil, nil, nil, subRepo, nil, nil, nil, nil)

	got, err := svc.normalizePreferredSubscriptionGroupID(ctx, &User{ID: 99}, &Group{
		ID:               routeGroupID,
		Name:             "Claude",
		Platform:         PlatformAnthropic,
		SubscriptionType: SubscriptionTypeSubscription,
		Status:           StatusActive,
	}, &routeGroupID)

	require.ErrorIs(t, err, ErrPreferredSubscriptionNotAllowed)
	require.Nil(t, got)
}
