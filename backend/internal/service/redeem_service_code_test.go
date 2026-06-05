//go:build unit

package service

import (
	"context"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type captureRedeemCodeRepo struct {
	createdBatch []RedeemCode
	codesByCode  map[string]*RedeemCode
	getByCodes   []string
	useCalls     []struct {
		id     int64
		userID int64
	}
}

func (r *captureRedeemCodeRepo) Create(context.Context, *RedeemCode) error {
	panic("unexpected Create call")
}
func (r *captureRedeemCodeRepo) CreateBatch(_ context.Context, codes []RedeemCode) error {
	r.createdBatch = append([]RedeemCode(nil), codes...)
	return nil
}
func (r *captureRedeemCodeRepo) GetByID(context.Context, int64) (*RedeemCode, error) {
	panic("unexpected GetByID call")
}
func (r *captureRedeemCodeRepo) GetByCode(_ context.Context, code string) (*RedeemCode, error) {
	r.getByCodes = append(r.getByCodes, code)
	if r.codesByCode != nil {
		if redeemCode, ok := r.codesByCode[code]; ok {
			copied := *redeemCode
			return &copied, nil
		}
	}
	return nil, ErrRedeemCodeNotFound
}
func (r *captureRedeemCodeRepo) Update(context.Context, *RedeemCode) error {
	panic("unexpected Update call")
}
func (r *captureRedeemCodeRepo) Delete(context.Context, int64) error { panic("unexpected Delete call") }
func (r *captureRedeemCodeRepo) Use(_ context.Context, id int64, userID int64) error {
	r.useCalls = append(r.useCalls, struct {
		id     int64
		userID int64
	}{id: id, userID: userID})
	return nil
}
func (r *captureRedeemCodeRepo) List(context.Context, pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (r *captureRedeemCodeRepo) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}
func (r *captureRedeemCodeRepo) ListByUser(context.Context, int64, int) ([]RedeemCode, error) {
	panic("unexpected ListByUser call")
}
func (r *captureRedeemCodeRepo) ListByUserPaginated(context.Context, int64, pagination.PaginationParams, string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserPaginated call")
}
func (r *captureRedeemCodeRepo) SumPositiveBalanceByUser(context.Context, int64) (float64, error) {
	panic("unexpected SumPositiveBalanceByUser call")
}

func TestRedeemService_GenerateRandomCodeFitsSchema(t *testing.T) {
	service := NewRedeemService(nil, nil, nil, nil, nil, nil, nil)

	code, err := service.GenerateRandomCode()

	require.NoError(t, err)
	require.Len(t, code, 32)
	require.NotContains(t, code, "-")
	require.Equal(t, code, strings.ToUpper(code))
}

func TestRedeemService_GenerateCodesInvitationFitsSchema(t *testing.T) {
	repo := &captureRedeemCodeRepo{}
	service := NewRedeemService(repo, nil, nil, nil, nil, nil, nil)

	codes, err := service.GenerateCodes(context.Background(), GenerateCodesRequest{
		Count: 2,
		Type:  RedeemTypeInvitation,
	})

	require.NoError(t, err)
	require.Len(t, codes, 2)
	require.Len(t, repo.createdBatch, 2)
	for _, code := range repo.createdBatch {
		require.Len(t, code.Code, 32)
		require.Equal(t, RedeemTypeInvitation, code.Type)
		require.Equal(t, StatusUnused, code.Status)
		require.Zero(t, code.Value)
	}
}
