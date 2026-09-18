package model

import (
	"encoding/json"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExemptionMatchTypeRoundTrip(t *testing.T) {
	for _, matchType := range []string{"exact", "prefix", "suffix"} {
		t.Run(matchType, func(t *testing.T) {
			input := Exemption{ID: "existing-exemption", Metric: "rpc_", MatchType: matchType, KeepLabels: []string{"method", "status"}}
			wire, err := json.Marshal(input)
			require.NoError(t, err)
			var decoded Exemption
			require.NoError(t, json.Unmarshal(wire, &decoded))
			state := decoded.ToTF()
			require.Equal(t, matchType, state.MatchType.ValueString())
			restored := state.ToAPIReq()
			require.Equal(t, input.ID, restored.ID)
			require.Equal(t, input.Metric, restored.Metric)
			require.Equal(t, input.MatchType, restored.MatchType)
			require.Equal(t, input.KeepLabels, restored.KeepLabels)
		})
	}
}

func TestLegacyExemptionDefaultsToExact(t *testing.T) {
	var input Exemption
	require.NoError(t, json.Unmarshal([]byte(`{"id":"legacy","metric":"rpc_requests_total"}`), &input))
	require.Equal(t, "exact", input.ToTF().MatchType.ValueString())
	require.Equal(t, "exact", input.ToTF().ToAPIReq().MatchType)
}

func TestExemptionExpiryPreserved(t *testing.T) {
	for _, expiry := range []string{"0001-01-01T00:00:00Z", "2026-12-01T12:34:56.123456Z"} {
		value := Exemption{ExpiresAt: &expiry}
		require.Equal(t, value.ExpiresAt, value.ToTF().ToAPIReq().ExpiresAt)
	}
	for _, v := range []types.String{types.StringNull(), types.StringUnknown()} {
		require.Nil(t, (ExemptionTF{ExpiresAt: v}).ToAPIReq().ExpiresAt)
	}
}
