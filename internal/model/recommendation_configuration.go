package model

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RecommendationConfiguration is a recommendation configuration as Adaptive Metrics serves it: the service keeps one,
// global-default, which every segment uses (its usage sources: dashboard, rules, queries). Every field it serves is
// kept, so an update replaces the usage sources and leaves the fields this provider does not manage as they were.
type RecommendationConfiguration struct {
	ID           string
	UsageSources []string
	fields       map[string]json.RawMessage
}

func (c *RecommendationConfiguration) UnmarshalJSON(body []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return err
	}
	var id string
	if err := json.Unmarshal(fields["id"], &id); err != nil || id == "" {
		return fmt.Errorf("recommendation configuration without an id")
	}
	var sources []string
	if raw, ok := fields["usage_sources"]; ok {
		if err := json.Unmarshal(raw, &sources); err != nil {
			return err
		}
	}
	*c = RecommendationConfiguration{ID: id, UsageSources: sources, fields: fields}
	return nil
}

func (c RecommendationConfiguration) MarshalJSON() ([]byte, error) {
	fields := map[string]json.RawMessage{}
	for k, v := range c.fields {
		fields[k] = v
	}
	var err error
	if fields["id"], err = json.Marshal(c.ID); err != nil {
		return nil, err
	}
	sources := c.UsageSources
	if sources == nil {
		sources = []string{}
	}
	if fields["usage_sources"], err = json.Marshal(sources); err != nil {
		return nil, err
	}
	return json.Marshal(fields)
}

type RecommendationConfigurationTF struct {
	ID           types.String   `tfsdk:"id"`
	UsageSources []types.String `tfsdk:"usage_sources"`
}

func (c RecommendationConfiguration) ToTF() RecommendationConfigurationTF {
	return RecommendationConfigurationTF{ID: types.StringValue(c.ID), UsageSources: toTypesStringSlice(c.UsageSources)}
}

func (c RecommendationConfigurationTF) ApplySettings(current RecommendationConfiguration) RecommendationConfiguration {
	current.UsageSources = toStringSlice(c.UsageSources)
	return current
}
