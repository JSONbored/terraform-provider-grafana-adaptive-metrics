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

func TestRecommendationConfigurationNativeLifecycle(t *testing.T) {
	for _, id := range []string{"global-default", "default", "01M22E09H808KGR3C27PNQ61BC"} {
		t.Run(id, func(t *testing.T) {
			timestamp := "2026-09-18T00:00:00Z"
			stored := model.RecommendationConfiguration{ID: id, Name: "existing", UsageSources: []string{"dashboard", "rules", "queries"}, UpdatedAt: &timestamp}
			reads, writes := 0, 0
			rejectWrite, unknownField := false, false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, "/aggregations/recommendation-configurations/"+id, req.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch req.Method {
				case http.MethodGet:
					reads++
					if unknownField {
						_, _ = w.Write([]byte(`{"id":"` + id + `","name":"existing","usage_sources":[],"future_setting":true}`))
						return
					}
					require.NoError(t, json.NewEncoder(w).Encode(stored))
				case http.MethodPut:
					writes++
					if rejectWrite {
						w.WriteHeader(http.StatusForbidden)
						return
					}
					var incoming model.RecommendationConfiguration
					require.NoError(t, json.NewDecoder(req.Body).Decode(&incoming))
					require.Equal(t, stored.ID, incoming.ID)
					require.Equal(t, stored.UpdatedAt, incoming.UpdatedAt)
					stored = incoming
					timestamp = "2026-09-18T00:01:00Z"
				default:
					t.Errorf("unexpected membership mutation %s", req.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			}))
			defer server.Close()
			c, err := client.New(server.URL, &client.Config{})
			require.NoError(t, err)
			r := &recommendationConfigurationResource{client: c}
			var sr resource.SchemaResponse
			r.Schema(t.Context(), resource.SchemaRequest{}, &sr)
			state := tfsdk.State{Schema: sr.Schema}
			require.False(t, state.Set(t.Context(), stored.ToTF()).HasError())
			imp := resource.ImportStateResponse{State: state}
			r.ImportState(t.Context(), resource.ImportStateRequest{ID: id}, &imp)
			require.False(t, imp.Diagnostics.HasError())
			read := resource.ReadResponse{State: imp.State}
			r.Read(t.Context(), resource.ReadRequest{State: imp.State}, &read)
			require.False(t, read.Diagnostics.HasError(), read.Diagnostics)
			var tf model.RecommendationConfigurationTF
			require.False(t, read.State.Get(t.Context(), &tf).HasError())
			require.Equal(t, stored.ToTF(), tf)
			require.Equal(t, 0, writes)
			// Remote usage changes must become visible to Terraform's next plan.
			stored.UsageSources = []string{"rules"}
			drift := resource.ReadResponse{State: read.State}
			r.Read(t.Context(), resource.ReadRequest{State: read.State}, &drift)
			require.False(t, drift.Diagnostics.HasError())
			require.False(t, drift.State.Get(t.Context(), &tf).HasError())
			require.Equal(t, []types.String{types.StringValue("rules")}, tf.UsageSources)
			// Explicit empty usage sources survive; IDs and server timestamps are retained.
			tf.UsageSources = []types.String{}
			tf.Name = types.StringValue("managed")
			planned := tfsdk.State{Schema: sr.Schema}
			require.False(t, planned.Set(t.Context(), tf).HasError())
			update := resource.UpdateResponse{State: drift.State}
			r.Update(t.Context(), resource.UpdateRequest{State: drift.State, Plan: tfsdk.Plan{Schema: sr.Schema, Raw: planned.Raw}}, &update)
			require.False(t, update.Diagnostics.HasError(), update.Diagnostics)
			require.Equal(t, 1, writes)
			require.Empty(t, stored.UsageSources)
			require.Equal(t, "managed", stored.Name)
			// A failed write leaves Terraform's prior state intact.
			rejectWrite = true
			failed := resource.UpdateResponse{State: update.State}
			r.Update(t.Context(), resource.UpdateRequest{State: update.State, Plan: tfsdk.Plan{Schema: sr.Schema, Raw: planned.Raw}}, &failed)
			require.True(t, failed.Diagnostics.HasError())
			require.True(t, failed.State.Raw.Equal(update.State.Raw))
			// Unknown server fields fail read instead of creating a lossy owner.
			unknownField = true
			bad := resource.ReadResponse{State: update.State}
			r.Read(t.Context(), resource.ReadRequest{State: update.State}, &bad)
			require.True(t, bad.Diagnostics.HasError())
			require.True(t, bad.State.Raw.Equal(update.State.Raw))
			// Relinquishing settings management never deletes the service-owned object.
			before := reads + writes
			var deleted resource.DeleteResponse
			r.Delete(t.Context(), resource.DeleteRequest{State: update.State}, &deleted)
			require.False(t, deleted.Diagnostics.HasError())
			require.Equal(t, before, reads+writes)
		})
	}
}

func TestRecommendationConfigurationUnknownParentCannotBeCreated(t *testing.T) {
	writes := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writes++
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()
	c, err := client.New(s.URL, &client.Config{})
	require.NoError(t, err)
	r := &recommendationConfigurationResource{client: c}
	var sr resource.SchemaResponse
	r.Schema(t.Context(), resource.SchemaRequest{}, &sr)
	cfg := model.RecommendationConfigurationTF{ID: types.StringValue("missing"), Name: types.StringValue("missing"), UsageSources: []types.String{}, UpdatedAt: types.StringUnknown()}
	state := tfsdk.State{Schema: sr.Schema}
	require.False(t, state.Set(t.Context(), cfg).HasError())
	response := resource.CreateResponse{State: state}
	r.Create(t.Context(), resource.CreateRequest{Plan: tfsdk.Plan{Schema: sr.Schema, Raw: state.Raw}}, &response)
	require.True(t, response.Diagnostics.HasError())
	require.Equal(t, 0, writes)
	_, err = c.ReadRecommendationConfiguration("../global-default")
	require.Error(t, err)
	require.Equal(t, 0, writes)
}
