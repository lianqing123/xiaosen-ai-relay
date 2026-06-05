package service

import (
	"context"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionplan"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// validatePlanRequired checks that all required fields for a plan are provided.
func validatePlanRequired(name string, groupID int64, price float64, validityDays int, validityUnit string, originalPrice *float64) error {
	if strings.TrimSpace(name) == "" {
		return infraerrors.BadRequest("PLAN_NAME_REQUIRED", "plan name is required")
	}
	if groupID <= 0 {
		return infraerrors.BadRequest("PLAN_GROUP_REQUIRED", "group is required")
	}
	if price <= 0 {
		return infraerrors.BadRequest("PLAN_PRICE_INVALID", "price must be > 0")
	}
	if validityDays <= 0 {
		return infraerrors.BadRequest("PLAN_VALIDITY_REQUIRED", "validity days must be > 0")
	}
	if _, ok := normalizePlanValidityUnit(validityUnit); !ok {
		return infraerrors.BadRequest("PLAN_VALIDITY_UNIT_REQUIRED", "validity unit must be day, week, month, or year")
	}
	if originalPrice != nil && *originalPrice < 0 {
		return infraerrors.BadRequest("PLAN_ORIGINAL_PRICE_INVALID", "original price must be >= 0")
	}
	return nil
}

// validatePlanPatch validates only the non-nil fields in a patch update.
func validatePlanPatch(req UpdatePlanRequest) error {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return infraerrors.BadRequest("PLAN_NAME_REQUIRED", "plan name is required")
	}
	if req.GroupID != nil && *req.GroupID <= 0 {
		return infraerrors.BadRequest("PLAN_GROUP_REQUIRED", "group is required")
	}
	if req.Price != nil && *req.Price <= 0 {
		return infraerrors.BadRequest("PLAN_PRICE_INVALID", "price must be > 0")
	}
	if req.ValidityDays != nil && *req.ValidityDays <= 0 {
		return infraerrors.BadRequest("PLAN_VALIDITY_REQUIRED", "validity days must be > 0")
	}
	if req.ValidityUnit != nil {
		if _, ok := normalizePlanValidityUnit(*req.ValidityUnit); !ok {
			return infraerrors.BadRequest("PLAN_VALIDITY_UNIT_REQUIRED", "validity unit must be day, week, month, or year")
		}
	}
	if req.OriginalPrice != nil && *req.OriginalPrice < 0 {
		return infraerrors.BadRequest("PLAN_ORIGINAL_PRICE_INVALID", "original price must be >= 0")
	}
	return nil
}

func normalizePlanSubscriptionGroupIDs(primaryGroupID int64, includedGroupIDs []int64) []int64 {
	seen := make(map[int64]bool, len(includedGroupIDs)+1)
	out := make([]int64, 0, len(includedGroupIDs)+1)
	if primaryGroupID > 0 {
		seen[primaryGroupID] = true
		out = append(out, primaryGroupID)
	}
	for _, groupID := range includedGroupIDs {
		if groupID <= 0 || seen[groupID] {
			continue
		}
		seen[groupID] = true
		out = append(out, groupID)
	}
	return out
}

// NormalizePlanSubscriptionGroupIDsForDisplay returns the full group list for a
// plan with the primary group first. It is exported for response shaping only.
func NormalizePlanSubscriptionGroupIDsForDisplay(primaryGroupID int64, includedGroupIDs []int64) []int64 {
	return normalizePlanSubscriptionGroupIDs(primaryGroupID, includedGroupIDs)
}

func planAdditionalGroupIDs(primaryGroupID int64, groupIDs []int64) []int64 {
	out := make([]int64, 0, len(groupIDs))
	seen := make(map[int64]bool, len(groupIDs))
	for _, groupID := range groupIDs {
		if groupID <= 0 || groupID == primaryGroupID || seen[groupID] {
			continue
		}
		seen[groupID] = true
		out = append(out, groupID)
	}
	return out
}

func normalizePlanValidityUnit(unit string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case validityUnitDay, "days":
		return validityUnitDay, true
	case validityUnitWeek, "weeks":
		return validityUnitWeek, true
	case validityUnitMonth, "months":
		return validityUnitMonth, true
	case validityUnitYear, "years":
		return validityUnitYear, true
	default:
		return "", false
	}
}

// --- Plan CRUD ---

// PlanGroupInfo holds the group details needed for subscription plan display.
type PlanGroupInfo struct {
	Platform        string   `json:"platform"`
	Name            string   `json:"name"`
	RateMultiplier  float64  `json:"rate_multiplier"`
	DailyLimitUSD   *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD  *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD *float64 `json:"monthly_limit_usd"`
	ModelScopes     []string `json:"supported_model_scopes"`
}

// GetGroupPlatformMap returns a map of group_id → platform for the given plans.
func (s *PaymentConfigService) GetGroupPlatformMap(ctx context.Context, plans []*dbent.SubscriptionPlan) map[int64]string {
	info := s.GetGroupInfoMap(ctx, plans)
	m := make(map[int64]string, len(info))
	for id, gi := range info {
		m[id] = gi.Platform
	}
	return m
}

// GetGroupInfoMap returns a map of group_id → PlanGroupInfo for the given plans.
func (s *PaymentConfigService) GetGroupInfoMap(ctx context.Context, plans []*dbent.SubscriptionPlan) map[int64]PlanGroupInfo {
	ids := make([]int64, 0, len(plans))
	seen := make(map[int64]bool)
	for _, p := range plans {
		for _, id := range normalizePlanSubscriptionGroupIDs(p.GroupID, p.IncludedGroupIds) {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	groups, err := s.entClient.Group.Query().Where(group.IDIn(ids...)).All(ctx)
	if err != nil {
		return nil
	}
	m := make(map[int64]PlanGroupInfo, len(groups))
	for _, g := range groups {
		m[int64(g.ID)] = PlanGroupInfo{
			Platform:        g.Platform,
			Name:            g.Name,
			RateMultiplier:  g.RateMultiplier,
			DailyLimitUSD:   g.DailyLimitUsd,
			WeeklyLimitUSD:  g.WeeklyLimitUsd,
			MonthlyLimitUSD: g.MonthlyLimitUsd,
			ModelScopes:     g.SupportedModelScopes,
		}
	}
	return m
}

func (s *PaymentConfigService) ListPlans(ctx context.Context) ([]*dbent.SubscriptionPlan, error) {
	return s.entClient.SubscriptionPlan.Query().Order(subscriptionplan.BySortOrder()).All(ctx)
}

func (s *PaymentConfigService) ListPlansForSale(ctx context.Context) ([]*dbent.SubscriptionPlan, error) {
	return s.entClient.SubscriptionPlan.Query().Where(subscriptionplan.ForSaleEQ(true)).Order(subscriptionplan.BySortOrder()).All(ctx)
}

func (s *PaymentConfigService) CreatePlan(ctx context.Context, req CreatePlanRequest) (*dbent.SubscriptionPlan, error) {
	if err := validatePlanRequired(req.Name, req.GroupID, req.Price, req.ValidityDays, req.ValidityUnit, req.OriginalPrice); err != nil {
		return nil, err
	}
	includedGroupIDs, err := s.validatePlanSubscriptionGroups(ctx, req.GroupID, req.IncludedGroupIDs)
	if err != nil {
		return nil, err
	}
	validityUnit, _ := normalizePlanValidityUnit(req.ValidityUnit)
	b := s.entClient.SubscriptionPlan.Create().
		SetGroupID(req.GroupID).SetIncludedGroupIds(includedGroupIDs).
		SetName(req.Name).SetDescription(req.Description).
		SetPrice(req.Price).SetValidityDays(req.ValidityDays).SetValidityUnit(validityUnit).
		SetFeatures(req.Features).SetProductName(req.ProductName).
		SetForSale(req.ForSale).SetSortOrder(req.SortOrder)
	if req.OriginalPrice != nil {
		b.SetOriginalPrice(*req.OriginalPrice)
	}
	return b.Save(ctx)
}

// UpdatePlan updates a subscription plan by ID (patch semantics).
// NOTE: This function exceeds 30 lines due to per-field nil-check patch update boilerplate
// plus a validation guard for non-nil fields.
func (s *PaymentConfigService) UpdatePlan(ctx context.Context, id int64, req UpdatePlanRequest) (*dbent.SubscriptionPlan, error) {
	if err := validatePlanPatch(req); err != nil {
		return nil, err
	}
	if req.GroupID != nil || req.IncludedGroupIDs != nil {
		current, err := s.GetPlan(ctx, id)
		if err != nil {
			return nil, err
		}
		nextGroupID := current.GroupID
		if req.GroupID != nil {
			nextGroupID = *req.GroupID
		}
		nextIncludedGroupIDs := current.IncludedGroupIds
		if req.IncludedGroupIDs != nil {
			nextIncludedGroupIDs = *req.IncludedGroupIDs
		}
		normalizedIncludedGroupIDs, err := s.validatePlanSubscriptionGroups(ctx, nextGroupID, nextIncludedGroupIDs)
		if err != nil {
			return nil, err
		}
		req.IncludedGroupIDs = &normalizedIncludedGroupIDs
	}
	u := s.entClient.SubscriptionPlan.UpdateOneID(id)
	if req.GroupID != nil {
		u.SetGroupID(*req.GroupID)
	}
	if req.IncludedGroupIDs != nil {
		u.SetIncludedGroupIds(*req.IncludedGroupIDs)
	}
	if req.Name != nil {
		u.SetName(*req.Name)
	}
	if req.Description != nil {
		u.SetDescription(*req.Description)
	}
	if req.Price != nil {
		u.SetPrice(*req.Price)
	}
	if req.OriginalPrice != nil {
		u.SetOriginalPrice(*req.OriginalPrice)
	}
	if req.ValidityDays != nil {
		u.SetValidityDays(*req.ValidityDays)
	}
	if req.ValidityUnit != nil {
		validityUnit, _ := normalizePlanValidityUnit(*req.ValidityUnit)
		u.SetValidityUnit(validityUnit)
	}
	if req.Features != nil {
		u.SetFeatures(*req.Features)
	}
	if req.ProductName != nil {
		u.SetProductName(*req.ProductName)
	}
	if req.ForSale != nil {
		u.SetForSale(*req.ForSale)
	}
	if req.SortOrder != nil {
		u.SetSortOrder(*req.SortOrder)
	}
	return u.Save(ctx)
}

func (s *PaymentConfigService) DeletePlan(ctx context.Context, id int64) error {
	count, err := s.countPendingOrdersByPlan(ctx, id)
	if err != nil {
		return fmt.Errorf("check pending orders: %w", err)
	}
	if count > 0 {
		return infraerrors.Conflict("PENDING_ORDERS",
			fmt.Sprintf("this plan has %d in-progress orders and cannot be deleted — wait for orders to complete first", count))
	}
	return s.entClient.SubscriptionPlan.DeleteOneID(id).Exec(ctx)
}

// GetPlan returns a subscription plan by ID.
func (s *PaymentConfigService) GetPlan(ctx context.Context, id int64) (*dbent.SubscriptionPlan, error) {
	plan, err := s.entClient.SubscriptionPlan.Get(ctx, id)
	if err != nil {
		return nil, infraerrors.NotFound("PLAN_NOT_FOUND", "subscription plan not found")
	}
	return plan, nil
}

func (s *PaymentConfigService) validatePlanSubscriptionGroups(ctx context.Context, primaryGroupID int64, includedGroupIDs []int64) ([]int64, error) {
	groupIDs := normalizePlanSubscriptionGroupIDs(primaryGroupID, includedGroupIDs)
	if len(groupIDs) == 0 {
		return nil, infraerrors.BadRequest("PLAN_GROUP_REQUIRED", "group is required")
	}

	groups, err := s.entClient.Group.Query().Where(group.IDIn(groupIDs...)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("load plan groups: %w", err)
	}
	byID := make(map[int64]*dbent.Group, len(groups))
	for _, g := range groups {
		byID[int64(g.ID)] = g
	}
	for _, groupID := range groupIDs {
		g := byID[groupID]
		if g == nil {
			return nil, infraerrors.NotFound("GROUP_NOT_FOUND", fmt.Sprintf("subscription group %d is not available", groupID))
		}
		if g.Status != StatusActive {
			return nil, infraerrors.NotFound("GROUP_NOT_FOUND", fmt.Sprintf("subscription group %d is inactive", groupID))
		}
		if g.SubscriptionType != SubscriptionTypeSubscription {
			return nil, infraerrors.BadRequest("GROUP_TYPE_MISMATCH", fmt.Sprintf("group %d is not a subscription type", groupID))
		}
	}
	return planAdditionalGroupIDs(primaryGroupID, groupIDs), nil
}
