package service

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionplan"
)

func subscriptionRouteIncludedByActiveUserPlan(ctx context.Context, repo UserSubscriptionRepository, entClient *dbent.Client, userID, routeGroupID int64) (bool, error) {
	if repo == nil || entClient == nil || userID <= 0 || routeGroupID <= 0 {
		return false, nil
	}
	activeSubscriptions, err := repo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	subscribedGroupIDs := make(map[int64]bool, len(activeSubscriptions))
	for _, sub := range activeSubscriptions {
		if sub.GroupID > 0 {
			subscribedGroupIDs[sub.GroupID] = true
		}
	}
	includedRouteGroupIDs, err := includedRouteGroupIDsForSubscribedGroups(ctx, entClient, subscribedGroupIDs)
	if err != nil {
		return false, err
	}
	return includedRouteGroupIDs[routeGroupID], nil
}

func includedRouteGroupIDsForSubscribedGroups(ctx context.Context, entClient *dbent.Client, subscribedGroupIDs map[int64]bool) (map[int64]bool, error) {
	out := make(map[int64]bool)
	if entClient == nil || len(subscribedGroupIDs) == 0 {
		return out, nil
	}
	primaryGroupIDs := make([]int64, 0, len(subscribedGroupIDs))
	for groupID := range subscribedGroupIDs {
		if groupID > 0 {
			primaryGroupIDs = append(primaryGroupIDs, groupID)
		}
	}
	if len(primaryGroupIDs) == 0 {
		return out, nil
	}
	plans, err := entClient.SubscriptionPlan.Query().
		Where(subscriptionplan.GroupIDIn(primaryGroupIDs...)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, plan := range plans {
		if plan == nil {
			continue
		}
		for _, includedGroupID := range plan.IncludedGroupIds {
			if includedGroupID <= 0 || includedGroupID == plan.GroupID {
				continue
			}
			out[includedGroupID] = true
		}
	}
	return out, nil
}
