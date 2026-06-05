package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type affiliateTransferRepoStub struct {
	userID        int64
	multiplier    float64
	transferred   float64
	balance       float64
	transferError error
}

type affiliateSettingRepoStub struct {
	values map[string]string
}

func (s *affiliateSettingRepoStub) Get(_ context.Context, key string) (*Setting, error) {
	return &Setting{Key: key, Value: s.values[key]}, nil
}
func (s *affiliateSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}
func (s *affiliateSettingRepoStub) Set(context.Context, string, string) error {
	return nil
}
func (s *affiliateSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		result[key] = s.values[key]
	}
	return result, nil
}
func (s *affiliateSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}
func (s *affiliateSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}
func (s *affiliateSettingRepoStub) Delete(context.Context, string) error {
	return nil
}

func (s *affiliateTransferRepoStub) EnsureUserAffiliate(context.Context, int64) (*AffiliateSummary, error) {
	panic("unexpected EnsureUserAffiliate call")
}
func (s *affiliateTransferRepoStub) GetAffiliateByCode(context.Context, string) (*AffiliateSummary, error) {
	panic("unexpected GetAffiliateByCode call")
}
func (s *affiliateTransferRepoStub) BindInviter(context.Context, int64, int64) (bool, error) {
	panic("unexpected BindInviter call")
}
func (s *affiliateTransferRepoStub) AccrueQuota(context.Context, int64, int64, float64, int) (bool, error) {
	panic("unexpected AccrueQuota call")
}
func (s *affiliateTransferRepoStub) GetAccruedRebateFromInvitee(context.Context, int64, int64) (float64, error) {
	panic("unexpected GetAccruedRebateFromInvitee call")
}
func (s *affiliateTransferRepoStub) ThawFrozenQuota(context.Context, int64) (float64, error) {
	panic("unexpected ThawFrozenQuota call")
}
func (s *affiliateTransferRepoStub) TransferQuotaToBalance(_ context.Context, userID int64, multiplier float64) (float64, float64, error) {
	s.userID = userID
	s.multiplier = multiplier
	return s.transferred, s.balance, s.transferError
}
func (s *affiliateTransferRepoStub) ListInvitees(context.Context, int64, int) ([]AffiliateInvitee, error) {
	panic("unexpected ListInvitees call")
}
func (s *affiliateTransferRepoStub) UpdateUserAffCode(context.Context, int64, string) error {
	panic("unexpected UpdateUserAffCode call")
}
func (s *affiliateTransferRepoStub) ResetUserAffCode(context.Context, int64) (string, error) {
	panic("unexpected ResetUserAffCode call")
}
func (s *affiliateTransferRepoStub) SetUserRebateRate(context.Context, int64, *float64) error {
	panic("unexpected SetUserRebateRate call")
}
func (s *affiliateTransferRepoStub) BatchSetUserRebateRate(context.Context, []int64, *float64) error {
	panic("unexpected BatchSetUserRebateRate call")
}
func (s *affiliateTransferRepoStub) ListUsersWithCustomSettings(context.Context, AffiliateAdminFilter) ([]AffiliateAdminEntry, int64, error) {
	panic("unexpected ListUsersWithCustomSettings call")
}
func (s *affiliateTransferRepoStub) ListInviteeRebates(context.Context, AffiliateInviteeRebateFilter) ([]AffiliateInviteeRebateEntry, int64, error) {
	panic("unexpected ListInviteeRebates call")
}

func TestTransferAffiliateQuota_UsesPaymentRechargeMultiplier(t *testing.T) {
	t.Parallel()

	repo := &affiliateTransferRepoStub{transferred: 12, balance: 35}
	settingSvc := NewSettingService(&affiliateSettingRepoStub{values: map[string]string{
		SettingBalanceRechargeMult: "2.5",
	}}, &config.Config{})
	svc := NewAffiliateService(repo, settingSvc, nil, nil)

	transferred, balance, err := svc.TransferAffiliateQuota(context.Background(), 42)
	require.NoError(t, err)
	require.InDelta(t, 12, transferred, 1e-9)
	require.InDelta(t, 35, balance, 1e-9)
	require.Equal(t, int64(42), repo.userID)
	require.InDelta(t, 2.5, repo.multiplier, 1e-9)
}
