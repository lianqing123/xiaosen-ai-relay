//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type paymentAuthInvalidatorRecorder struct {
	keys []string
}

func (r *paymentAuthInvalidatorRecorder) InvalidateAuthCacheByKey(_ context.Context, key string) {
	r.keys = append(r.keys, key)
}

func (r *paymentAuthInvalidatorRecorder) InvalidateAuthCacheByUserID(context.Context, int64) {}

func (r *paymentAuthInvalidatorRecorder) InvalidateAuthCacheByGroupID(context.Context, int64) {}

func TestEnsureSubscriptionAPIKeyAccessMigratesStandardKeysToSubscriptionGroup(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	user, err := client.User.Create().
		SetEmail("subscriber@example.com").
		SetPasswordHash("hash").
		SetUsername("subscriber").
		Save(ctx)
	require.NoError(t, err)

	standardGroup, err := client.Group.Create().
		SetName("Standard").
		SetPlatform(PlatformOpenAI).
		SetSubscriptionType(SubscriptionTypeStandard).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	subscriptionGroup, err := client.Group.Create().
		SetName("Codex Standard").
		SetPlatform(PlatformOpenAI).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	const existingKey = "sk-existing-subscription-user"
	apiKey, err := client.APIKey.Create().
		SetUserID(user.ID).
		SetKey(existingKey).
		SetName("default").
		SetStatus(StatusActive).
		SetGroupID(standardGroup.ID).
		Save(ctx)
	require.NoError(t, err)

	recorder := &paymentAuthInvalidatorRecorder{}
	svc := &PaymentService{entClient: client}
	svc.SetAuthCacheInvalidator(recorder)

	err = svc.ensureSubscriptionAPIKeyAccess(ctx, user.ID, subscriptionGroup.ID)
	require.NoError(t, err)

	reloaded, err := client.APIKey.Get(ctx, apiKey.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.GroupID)
	require.Equal(t, subscriptionGroup.ID, *reloaded.GroupID)
	require.Equal(t, []string{existingKey}, recorder.keys)
}

func TestEnsureSubscriptionAPIKeyAccessDoesNotMoveOtherSubscriptionKeys(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	user, err := client.User.Create().
		SetEmail("multi-sub@example.com").
		SetPasswordHash("hash").
		SetUsername("multi-sub").
		Save(ctx)
	require.NoError(t, err)

	otherSubscriptionGroup, err := client.Group.Create().
		SetName("Other Subscription").
		SetPlatform(PlatformOpenAI).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	newSubscriptionGroup, err := client.Group.Create().
		SetName("New Subscription").
		SetPlatform(PlatformOpenAI).
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	const existingKey = "sk-existing-other-subscription"
	apiKey, err := client.APIKey.Create().
		SetUserID(user.ID).
		SetKey(existingKey).
		SetName("other-sub").
		SetStatus(StatusActive).
		SetGroupID(otherSubscriptionGroup.ID).
		Save(ctx)
	require.NoError(t, err)

	recorder := &paymentAuthInvalidatorRecorder{}
	svc := &PaymentService{entClient: client}
	svc.SetAuthCacheInvalidator(recorder)

	err = svc.ensureSubscriptionAPIKeyAccess(ctx, user.ID, newSubscriptionGroup.ID)
	require.NoError(t, err)

	reloaded, err := client.APIKey.Get(ctx, apiKey.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.GroupID)
	require.Equal(t, otherSubscriptionGroup.ID, *reloaded.GroupID)
	require.Empty(t, recorder.keys)
}
