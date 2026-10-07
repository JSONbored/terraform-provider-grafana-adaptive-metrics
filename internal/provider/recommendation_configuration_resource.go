package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-grafana-adaptive-metrics/internal/client"
	"github.com/hashicorp/terraform-provider-grafana-adaptive-metrics/internal/model"
)

type recommendationConfigurationResource struct{ client *client.Client }

var _ resource.ResourceWithImportState = &recommendationConfigurationResource{}
var _ resource.ResourceWithConfigure = &recommendationConfigurationResource{}

func newRecommendationConfigurationResource() resource.Resource {
	return &recommendationConfigurationResource{}
}
func (r *recommendationConfigurationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*resourceData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider configuration", fmt.Sprintf("Got %T", req.ProviderData))
		return
	}
	r.client = data.client
}
func (r *recommendationConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recommendation_configuration"
}
func (r *recommendationConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Manage the usage sources of an existing recommendation configuration: the service keeps one, global-default, which every segment uses. Import by its ID; destroy relinquishes settings management without deleting the service-owned object. Use prevent_destroy to retain Terraform ownership.", Attributes: map[string]schema.Attribute{
		"id":            schema.StringAttribute{Required: true, Description: "Existing configuration ID (global-default).", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"usage_sources": schema.ListAttribute{Required: true, ElementType: types.StringType, Description: "Usage sources recommendations count: dashboard, rules (always included by the service) and queries."},
	}}
}
func (r *recommendationConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
func (r *recommendationConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model.RecommendationConfigurationTF
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.ReadRecommendationConfiguration(state.ID.ValueString())
	if err != nil {
		if client.IsErrNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read recommendation configuration", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg.ToTF())...)
}
func (r *recommendationConfigurationResource) apply(ctx context.Context, plan model.RecommendationConfigurationTF) (model.RecommendationConfiguration, error) {
	current, err := r.client.ReadRecommendationConfiguration(plan.ID.ValueString())
	if err != nil {
		return current, err
	}
	desired := plan.ApplySettings(current)
	if err = r.client.UpdateRecommendationConfiguration(desired); err != nil {
		return current, err
	}
	return r.client.ReadRecommendationConfiguration(plan.ID.ValueString())
}
func (r *recommendationConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model.RecommendationConfigurationTF
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.apply(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to manage existing recommendation configuration", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg.ToTF())...)
	resp.Diagnostics.AddWarning("Existing configuration settings adopted", "The configuration is the service's. No new configuration object was created.")
}
func (r *recommendationConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model.RecommendationConfigurationTF
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.apply(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update recommendation configuration", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg.ToTF())...)
}
func (r *recommendationConfigurationResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Configuration settings no longer managed", "The service-owned configuration and its settings are preserved. Terraform has relinquished management; use prevent_destroy to require explicit ownership transfer.")
}
