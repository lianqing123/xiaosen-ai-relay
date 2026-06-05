//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizePlanValidityUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{name: "day", raw: "day", want: "day", ok: true},
		{name: "legacy days", raw: "days", want: "day", ok: true},
		{name: "week", raw: "week", want: "week", ok: true},
		{name: "legacy weeks", raw: "weeks", want: "week", ok: true},
		{name: "month", raw: "month", want: "month", ok: true},
		{name: "legacy months", raw: "months", want: "month", ok: true},
		{name: "year", raw: "year", want: "year", ok: true},
		{name: "legacy years", raw: "years", want: "year", ok: true},
		{name: "trim case", raw: "  MONTH  ", want: "month", ok: true},
		{name: "invalid", raw: "quarter", want: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizePlanValidityUnit(tt.raw)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPsComputeValidityDaysSupportsCanonicalAndLegacyUnits(t *testing.T) {
	t.Parallel()

	require.Equal(t, 30, psComputeValidityDays(30, "day"))
	require.Equal(t, 30, psComputeValidityDays(30, "days"))
	require.Equal(t, 14, psComputeValidityDays(2, "week"))
	require.Equal(t, 14, psComputeValidityDays(2, "weeks"))
	require.Equal(t, 60, psComputeValidityDays(2, "month"))
	require.Equal(t, 60, psComputeValidityDays(2, "months"))
	require.Equal(t, 365, psComputeValidityDays(1, "year"))
	require.Equal(t, 730, psComputeValidityDays(2, "years"))
}

func TestValidatePlanRequiredRejectsUnknownValidityUnit(t *testing.T) {
	t.Parallel()

	err := validatePlanRequired("Pro", 1, 9.99, 1, "quarter", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "validity unit")
}

func TestValidatePlanPatchRejectsUnknownValidityUnit(t *testing.T) {
	t.Parallel()

	unit := "quarter"
	err := validatePlanPatch(UpdatePlanRequest{ValidityUnit: &unit})
	require.Error(t, err)
	require.Contains(t, err.Error(), "validity unit")
}
