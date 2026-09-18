package model

import (
	"bytes"
	"encoding/json"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RecommendationConfiguration contains the settings of an existing tenant or
// segment configuration. Membership is owned by the service/segment lifecycle.
type RecommendationConfiguration struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	UsageSources []string `json:"usage_sources"`
	UpdatedAt    *string  `json:"updated_at,omitempty"`
}

// Reject unknown fields instead of silently dropping live settings on update.
func (c *RecommendationConfiguration) UnmarshalJSON(body []byte) error {
	type wire RecommendationConfiguration
	var value wire
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*c = RecommendationConfiguration(value)
	return nil
}

type RecommendationConfigurationTF struct {
	ID           types.String   `tfsdk:"id"`
	Name         types.String   `tfsdk:"name"`
	UsageSources []types.String `tfsdk:"usage_sources"`
	UpdatedAt    types.String   `tfsdk:"updated_at"`
}

func (c RecommendationConfiguration) ToTF() RecommendationConfigurationTF {
	return RecommendationConfigurationTF{ID: types.StringValue(c.ID), Name: types.StringValue(c.Name), UsageSources: toTypesStringSlice(c.UsageSources), UpdatedAt: types.StringPointerValue(c.UpdatedAt)}
}

func (c RecommendationConfigurationTF) ApplySettings(current RecommendationConfiguration) RecommendationConfiguration {
	current.Name = c.Name.ValueString()
	current.UsageSources = toStringSlice(c.UsageSources)
	return current
}
