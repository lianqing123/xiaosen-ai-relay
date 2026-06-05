//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUsageUnrestrictedUsesBillingSubscriptionGroupForClaudeRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/usage", nil)

	dailyLimit := 10.0
	monthlyLimit := 100.0
	routeGroup := &service.Group{
		ID:               200,
		Name:             "Claude",
		Platform:         service.PlatformAnthropic,
		SubscriptionType: service.SubscriptionTypeSubscription,
		Status:           service.StatusActive,
	}
	billingGroup := &service.Group{
		ID:               100,
		Name:             "Codex Pro",
		Platform:         service.PlatformOpenAI,
		SubscriptionType: service.SubscriptionTypeSubscription,
		Status:           service.StatusActive,
		DailyLimitUSD:    &dailyLimit,
		MonthlyLimitUSD:  &monthlyLimit,
	}
	subscription := &service.UserSubscription{
		ID:              300,
		UserID:          400,
		GroupID:         billingGroup.ID,
		Status:          service.SubscriptionStatusActive,
		DailyUsageUSD:   3,
		MonthlyUsageUSD: 60,
		ExpiresAt:       time.Now().Add(24 * time.Hour),
		Group:           billingGroup,
	}
	c.Set(string(middleware2.ContextKeySubscription), subscription)

	h := &GatewayHandler{}
	h.usageUnrestricted(
		c,
		context.Background(),
		&service.APIKey{
			ID:     500,
			UserID: subscription.UserID,
			Status: service.StatusAPIKeyActive,
			User:   &service.User{ID: subscription.UserID},
			Group:  routeGroup,
		},
		middleware2.AuthSubject{UserID: subscription.UserID},
		gin.H{"total_requests": 2},
		gin.H{"claude-sonnet-4-5": gin.H{"requests": 2}},
	)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "Codex Pro", resp["planName"])
	require.Equal(t, 7.0, resp["remaining"])

	subPayload, ok := resp["subscription"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(billingGroup.ID), subPayload["group_id"])
	require.Equal(t, billingGroup.Name, subPayload["group_name"])
	require.Equal(t, float64(routeGroup.ID), subPayload["route_group_id"])
	require.Equal(t, routeGroup.Name, subPayload["route_group_name"])
	require.Equal(t, true, subPayload["shared_with_subscription"])
	require.Equal(t, 3.0, subPayload["daily_usage_usd"])
	require.Equal(t, 60.0, subPayload["monthly_usage_usd"])
	require.Equal(t, dailyLimit, subPayload["daily_limit_usd"])
	require.Equal(t, monthlyLimit, subPayload["monthly_limit_usd"])
}
