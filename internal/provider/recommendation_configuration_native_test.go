package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-grafana-adaptive-metrics/internal/client"
	"github.com/hashicorp/terraform-provider-grafana-adaptive-metrics/internal/model"
	"github.com/stretchr/testify/require"
)

// The service lists its configurations and replaces one by ID (Adaptive Metrics app 1.20.0: a GET of one by ID answers
// 405, and global-default is the only one, without a name).
func TestRecommendationConfigurationReadsTheListAndKeepsServedFields(t *testing.T) {
	stored := map[string]any{"id": "global-default", "usage_sources": []string{"dashboard", "rules", "queries"}, "future_setting": true}
	reads, writes, rejectWrite := 0, 0, false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/aggregations/recommendation-configurations":
			reads++
			require.NoError(t, json.NewEncoder(w).Encode([]any{stored}))
		case req.Method == http.MethodPut && req.URL.Path == "/aggregations/recommendation-configurations/global-default":
			writes++
			if rejectWrite {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			var incoming map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&incoming))
			require.Equal(t, "global-default", incoming["id"])
			require.Equal(t, true, incoming["future_setting"], "a field this provider does not manage is sent back as served")
			stored = incoming
		default:
			t.Errorf("unexpected %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	c, err := client.New(server.URL, &client.Config{})
	require.NoError(t, err)
	r := &recommendationConfigurationResource{client: c}
	var sr resource.SchemaResponse
	r.Schema(t.Context(), resource.SchemaRequest{}, &sr)

	imp := resource.ImportStateResponse{State: tfsdk.State{Schema: sr.Schema}}
	require.False(t, imp.State.Set(t.Context(), model.RecommendationConfigurationTF{ID: types.StringValue("global-default"), UsageSources: []types.String{}}).HasError())
	read := resource.ReadResponse{State: imp.State}
	r.Read(t.Context(), resource.ReadRequest{State: imp.State}, &read)
	require.False(t, read.Diagnostics.HasError(), read.Diagnostics)
	var tf model.RecommendationConfigurationTF
	require.False(t, read.State.Get(t.Context(), &tf).HasError())
	require.Equal(t, []types.String{types.StringValue("dashboard"), types.StringValue("rules"), types.StringValue("queries")}, tf.UsageSources)
	require.Equal(t, 0, writes)

	// An update replaces the usage sources alone.
	tf.UsageSources = []types.String{types.StringValue("rules"), types.StringValue("queries")}
	planned := tfsdk.State{Schema: sr.Schema}
	require.False(t, planned.Set(t.Context(), tf).HasError())
	update := resource.UpdateResponse{State: read.State}
	r.Update(t.Context(), resource.UpdateRequest{State: read.State, Plan: tfsdk.Plan{Schema: sr.Schema, Raw: planned.Raw}}, &update)
	require.False(t, update.Diagnostics.HasError(), update.Diagnostics)
	require.Equal(t, 1, writes)
	require.Equal(t, []any{"rules", "queries"}, stored["usage_sources"])

	// A failed write leaves Terraform's prior state intact.
	rejectWrite = true
	failed := resource.UpdateResponse{State: update.State}
	r.Update(t.Context(), resource.UpdateRequest{State: update.State, Plan: tfsdk.Plan{Schema: sr.Schema, Raw: planned.Raw}}, &failed)
	require.True(t, failed.Diagnostics.HasError())
	require.True(t, failed.State.Raw.Equal(update.State.Raw))

	// A configuration the service no longer lists leaves Terraform's state (the per-segment ones it retired).
	gone := tfsdk.State{Schema: sr.Schema}
	require.False(t, gone.Set(t.Context(), model.RecommendationConfigurationTF{ID: types.StringValue("01M22E09H808KGR3C27PNQ61BC"), UsageSources: []types.String{}}).HasError())
	missing := resource.ReadResponse{State: gone}
	r.Read(t.Context(), resource.ReadRequest{State: gone}, &missing)
	require.False(t, missing.Diagnostics.HasError(), missing.Diagnostics)
	require.True(t, missing.State.Raw.IsNull())

	// Relinquishing settings management never deletes the service-owned object.
	before := reads + writes
	var deleted resource.DeleteResponse
	r.Delete(t.Context(), resource.DeleteRequest{State: update.State}, &deleted)
	require.False(t, deleted.Diagnostics.HasError())
	require.Equal(t, before, reads+writes)

	_, err = c.ReadRecommendationConfiguration("../global-default")
	require.Error(t, err)
}
