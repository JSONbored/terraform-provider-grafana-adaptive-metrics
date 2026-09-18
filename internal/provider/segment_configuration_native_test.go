package provider

import (
	"encoding/json"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-grafana-adaptive-metrics/internal/client"
	"github.com/hashicorp/terraform-provider-grafana-adaptive-metrics/internal/model"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSegmentNativeImportRefreshAndUpdatePreserveConfiguration(t *testing.T) {
	configuration := "configuration-original"
	stored := model.Segment{ID: "segment", Name: "Ethereum", Selector: `{chain="eth"}`, FallbackToDefault: true, RecommendationConfigurationID: &configuration}
	updates := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/aggregations/rules/segments", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			require.NoError(t, json.NewEncoder(w).Encode([]model.Segment{stored}))
		case http.MethodPut:
			require.Equal(t, "segment", r.URL.Query().Get("segment"))
			updates++
			var incoming model.Segment
			require.NoError(t, json.NewDecoder(r.Body).Decode(&incoming))
			require.Equal(t, stored.RecommendationConfigurationID, incoming.RecommendationConfigurationID)
			require.Equal(t, stored.Selector, incoming.Selector)
			require.Equal(t, stored.AutoApply, incoming.AutoApply)
			stored = incoming
		default:
			t.Errorf("unexpected mutation %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	api, err := client.New(server.URL, &client.Config{})
	require.NoError(t, err)
	r := &segmentResource{client: api}
	var schema resource.SchemaResponse
	r.Schema(t.Context(), resource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	require.False(t, state.Set(t.Context(), stored.ToTF()).HasError())
	imported := resource.ImportStateResponse{State: state}
	r.ImportState(t.Context(), resource.ImportStateRequest{ID: "segment"}, &imported)
	require.False(t, imported.Diagnostics.HasError())
	read := resource.ReadResponse{State: imported.State}
	r.Read(t.Context(), resource.ReadRequest{State: imported.State}, &read)
	require.False(t, read.Diagnostics.HasError(), read.Diagnostics)
	var actual model.SegmentTF
	require.False(t, read.State.Get(t.Context(), &actual).HasError())
	require.Equal(t, "configuration-original", actual.RecommendationConfigurationID.ValueString())
	// A remote reference change must be surfaced by refresh, never hidden in old state.
	configuration = "configuration-remote"
	refreshed := resource.ReadResponse{State: read.State}
	r.Read(t.Context(), resource.ReadRequest{State: read.State}, &refreshed)
	require.False(t, refreshed.Diagnostics.HasError(), refreshed.Diagnostics)
	require.False(t, refreshed.State.Get(t.Context(), &actual).HasError())
	require.Equal(t, "configuration-remote", actual.RecommendationConfigurationID.ValueString())
	actual.Name = types.StringValue("Renamed Ethereum")
	planned := tfsdk.State{Schema: schema.Schema}
	require.False(t, planned.Set(t.Context(), &actual).HasError())
	updated := resource.UpdateResponse{State: refreshed.State}
	r.Update(t.Context(), resource.UpdateRequest{State: refreshed.State, Plan: tfsdk.Plan{Schema: schema.Schema, Raw: planned.Raw}}, &updated)
	require.False(t, updated.Diagnostics.HasError(), updated.Diagnostics)
	require.Equal(t, 1, updates)
	require.False(t, updated.State.Get(t.Context(), &actual).HasError())
	require.Equal(t, "configuration-remote", actual.RecommendationConfigurationID.ValueString())
	require.Equal(t, "Renamed Ethereum", actual.Name.ValueString())
}
