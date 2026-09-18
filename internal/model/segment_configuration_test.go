package model

import (
	"encoding/json"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSegmentRecommendationConfigurationRoundTrip(t *testing.T) {
	for _, body := range []string{
		`{"id":"segment","name":"Ethereum","selector":"{chain=\"eth\"}","fallback_to_default":true,"auto_apply":{"enabled":false},"recommendation_configuration_id":"configuration"}`,
		`{"id":"legacy","name":"Legacy","selector":"{chain=\"base\"}","fallback_to_default":false,"auto_apply":{"enabled":true}}`,
	} {
		var before Segment
		require.NoError(t, json.Unmarshal([]byte(body), &before))
		after := before.ToTF().ToAPIReq()
		require.Equal(t, before, after)
		encoded, err := json.Marshal(after)
		require.NoError(t, err)
		require.JSONEq(t, body, string(encoded))
	}
	for _, value := range []types.String{types.StringNull(), types.StringUnknown()} {
		result := (SegmentTF{RecommendationConfigurationID: value}).ToAPIReq()
		require.Nil(t, result.RecommendationConfigurationID)
	}
}
