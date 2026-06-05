//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanUserBindGroupInternalAllowsClaudeBalanceFallback(t *testing.T) {
	svc := &APIKeyService{}
	claudeGroup := &Group{
		ID:                   24,
		Name:                 "Claude",
		Platform:             PlatformAnthropic,
		SubscriptionType:     SubscriptionTypeSubscription,
		SupportedModelScopes: []string{"claude"},
	}
	openAIGroup := &Group{
		ID:               10,
		Name:             "Codex",
		Platform:         PlatformOpenAI,
		SubscriptionType: SubscriptionTypeSubscription,
	}

	require.True(t, svc.canUserBindGroupInternal(&User{Balance: 1}, claudeGroup, map[int64]bool{}, map[int64]bool{}))
	require.False(t, svc.canUserBindGroupInternal(&User{Balance: 0}, claudeGroup, map[int64]bool{}, map[int64]bool{}))
	require.True(t, svc.canUserBindGroupInternal(&User{Balance: 0}, claudeGroup, map[int64]bool{}, map[int64]bool{claudeGroup.ID: true}))
	require.False(t, svc.canUserBindGroupInternal(&User{Balance: 1}, openAIGroup, map[int64]bool{}, map[int64]bool{}))
	require.True(t, svc.canUserBindGroupInternal(&User{Balance: 0}, openAIGroup, map[int64]bool{openAIGroup.ID: true}, map[int64]bool{}))
}
