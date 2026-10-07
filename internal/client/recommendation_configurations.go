package client

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-provider-grafana-adaptive-metrics/internal/model"
)

// The service lists its configurations and replaces one by ID; it serves no single configuration by ID (a GET of
// .../<id> answers 405, Adaptive Metrics app 1.20.0, 2026-10-07).
const recommendationConfigurationsEndpoint = "/aggregations/recommendation-configurations"

var configurationIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func configurationPath(id string) (string, error) {
	if !configurationIDPattern.MatchString(id) {
		return "", fmt.Errorf("invalid recommendation configuration ID")
	}
	return recommendationConfigurationsEndpoint + "/" + id, nil
}

// ReadRecommendationConfiguration finds the configuration in the service's list; one it no longer lists is not found.
func (c *Client) ReadRecommendationConfiguration(id string) (model.RecommendationConfiguration, error) {
	if _, err := configurationPath(id); err != nil {
		return model.RecommendationConfiguration{}, err
	}
	var all []model.RecommendationConfiguration
	if err := c.request("GET", recommendationConfigurationsEndpoint, nil, nil, &all); err != nil {
		return model.RecommendationConfiguration{}, err
	}
	for _, config := range all {
		if config.ID == id {
			return config, nil
		}
	}
	return model.RecommendationConfiguration{}, ErrNotFound{BodyContents: []byte(fmt.Sprintf("recommendation configuration %q is not listed", id))}
}

func (c *Client) UpdateRecommendationConfiguration(config model.RecommendationConfiguration) error {
	path, err := configurationPath(config.ID)
	if err != nil {
		return err
	}
	body, err := json.Marshal(config)
	if err != nil {
		return err
	}
	c.segmentMutex.Lock()
	defer c.segmentMutex.Unlock()
	return c.request("PUT", path, nil, body, nil)
}
