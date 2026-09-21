package provider

import (
	"encoding/json"
	"fmt"
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

func TestRuleRefreshRemoteDeletionAndFailures(t *testing.T) {
	for _, segment := range []string{"", "segment-base"} {
		for _, status := range []int{200, 404, 401, 403, 429, 500} {
			t.Run(fmt.Sprintf("%s/%d", segment, status), func(t *testing.T) {
				stored := model.AggregationRule{Metric: "node_cpu_seconds_total", DropLabels: []string{"cpu"}, Aggregations: []string{"sum:counter", "count"}}
				requests := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					requests++
					require.Equal(t, http.MethodGet, req.Method)
					require.Equal(t, "/aggregations/rule/node_cpu_seconds_total", req.URL.Path)
					require.Equal(t, segment, req.URL.Query().Get("segment"))
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("ETag", "rules-version-2")
					w.WriteHeader(status)
					if status == http.StatusOK {
						require.NoError(t, json.NewEncoder(w).Encode(stored))
					} else {
						_, _ = w.Write([]byte(`{"error":"fixture response"}`))
					}
				}))
				defer server.Close()
				api, err := client.New(server.URL, &client.Config{})
				require.NoError(t, err)
				rules := NewAggregationRules(api)
				rules.segmentEtags = map[string]string{segment: "rules-version-1"}
				r := &ruleResource{rules: rules}
				var schema resource.SchemaResponse
				r.Schema(t.Context(), resource.SchemaRequest{}, &schema)
				state := tfsdk.State{Schema: schema.Schema}
				original := stored.ToTF()
				original.Segment = types.StringValue(segment)
				original.AutoImport = types.BoolValue(false)
				require.Empty(t, state.Set(t.Context(), &original))
				read := resource.ReadResponse{State: state}
				r.Read(t.Context(), resource.ReadRequest{State: state}, &read)
				require.Equal(t, 1, requests)
				switch status {
				case http.StatusNotFound:
					require.Empty(t, read.Diagnostics)
					require.True(t, read.State.Raw.IsNull())
				case http.StatusOK:
					require.Empty(t, read.Diagnostics)
					require.Equal(t, state.Raw, read.State.Raw)
					require.Equal(t, "rules-version-2", rules.segmentEtags[segment])
				default:
					require.True(t, read.Diagnostics.HasError())
					require.Equal(t, state.Raw, read.State.Raw)
				}
			})
		}
	}
}
