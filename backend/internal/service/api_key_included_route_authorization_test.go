//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type apiKeyIncludedRouteUserRepoStub struct {
	userRepoStubForGroupUpdate
	user *User
}

func (s *apiKeyIncludedRouteUserRepoStub) GetByID(_ context.Context, _ int64) (*User, error) {
	cp := *s.user
	return &cp, nil
}

type apiKeyIncludedRouteGroupRepoStub struct {
	groupRepoNoop
	groups []Group
}

func (s *apiKeyIncludedRouteGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	out := make([]Group, 0, len(s.groups))
	for _, group := range s.groups {
		if group.IsActive() {
			out = append(out, group)
		}
	}
	return out, nil
}

type apiKeyIncludedRouteSubRepoStub struct {
	userSubRepoNoop
	subs []UserSubscription
}

func (s *apiKeyIncludedRouteSubRepoStub) GetActiveByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*UserSubscription, error) {
	for _, sub := range s.subs {
		if sub.UserID == userID && sub.GroupID == groupID && sub.IsActive() {
			cp := sub
			return &cp, nil
		}
	}
	return nil, ErrSubscriptionNotFound
}

func (s *apiKeyIncludedRouteSubRepoStub) ListActiveByUserID(_ context.Context, userID int64) ([]UserSubscription, error) {
	out := make([]UserSubscription, 0, len(s.subs))
	for _, sub := range s.subs {
		if sub.UserID == userID && sub.IsActive() {
			out = append(out, sub)
		}
	}
	return out, nil
}

func TestAPIKeyServiceAllowsIncludedSubscriptionRoute(t *testing.T) {
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
		SetPrice(199).
		SetValidityDays(30).
		Save(ctx)
	require.NoError(t, err)

	user := &User{ID: 99, Balance: 0}
	route := Group{
		ID:                   routeGroup.ID,
		Name:                 "Claude",
		Platform:             PlatformAnthropic,
		Status:               StatusActive,
		SubscriptionType:     SubscriptionTypeSubscription,
		SupportedModelScopes: []string{"claude"},
	}
	primary := Group{
		ID:               primaryGroup.ID,
		Name:             "Codex Pro",
		Platform:         PlatformOpenAI,
		Status:           StatusActive,
		SubscriptionType: SubscriptionTypeSubscription,
	}
	subRepo := &apiKeyIncludedRouteSubRepoStub{subs: []UserSubscription{{
		ID:        1,
		UserID:    user.ID,
		GroupID:   primary.ID,
		Group:     &primary,
		Status:    StatusActive,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}}}
	svc := NewAPIKeyService(
		nil,
		&apiKeyIncludedRouteUserRepoStub{user: user},
		&apiKeyIncludedRouteGroupRepoStub{groups: []Group{route, primary}},
		subRepo,
		nil,
		nil,
		client,
		nil,
	)

	require.True(t, svc.canUserBindGroup(ctx, user, &route), "active primary package that includes Claude should allow binding Claude")

	groups, err := svc.GetAvailableGroups(ctx, user.ID)
	require.NoError(t, err)
	require.Contains(t, groupIDs(groups), route.ID, "Claude should be returned to the API Key group selector")
}

func groupIDs(groups []Group) []int64 {
	out := make([]int64, 0, len(groups))
	for _, group := range groups {
		out = append(out, group.ID)
	}
	return out
}
