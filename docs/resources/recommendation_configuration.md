---
page_title: "grafana-adaptive-metrics_recommendation_configuration Resource - grafana-adaptive-metrics"
subcategory: ""
description: |-
  Manage the usage sources of an existing recommendation configuration.
---

# grafana-adaptive-metrics_recommendation_configuration (Resource)

Manage the usage sources of an existing recommendation configuration: the service keeps one, `global-default`, which
every segment uses. It is read from the service's configurations list and replaced by ID; every field the service
serves besides `usage_sources` is sent back as served. Import by its ID. Destroy relinquishes settings management
without deleting the service-owned object; use `prevent_destroy` to retain Terraform ownership.

## Schema

### Required

- `id` (String) Existing configuration ID (`global-default`).
- `usage_sources` (List of String) Usage sources recommendations count: `dashboard`, `rules` (always included by the service) and `queries`.
