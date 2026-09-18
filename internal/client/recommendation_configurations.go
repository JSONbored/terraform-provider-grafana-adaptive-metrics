package client

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-provider-grafana-adaptive-metrics/internal/model"
	"regexp"
)

const recommendationConfigurationsEndpoint = "/aggregations/recommendation-configurations"

var configurationIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func configurationPath(id string) (string, error) {
	if !configurationIDPattern.MatchString(id) {
		return "", fmt.Errorf("invalid recommendation configuration ID")
	}
	return recommendationConfigurationsEndpoint + "/" + id, nil
}

func (c *Client) ReadRecommendationConfiguration(id string) (model.RecommendationConfiguration, error) {
	var config model.RecommendationConfiguration
	path, err := configurationPath(id)
	if err != nil {
		return config, err
	}
	err = c.request("GET", path, nil, nil, &config)
	if err == nil && config.ID != id {
		return config, fmt.Errorf("recommendation configuration response identity differs from request")
	}
	return config, err
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
	c.policyMutex.Lock()
	defer c.policyMutex.Unlock()
	return c.request("PUT", path, nil, body, nil)
}
