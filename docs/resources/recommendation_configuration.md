# Recommendation configuration settings

The SubnetOps native provider backport adds `grafana-adaptive-metrics_recommendation_configuration` for the current recommendation-configuration API. It manages every returned setting of an existing tenant or segment configuration: `id`, `name`, `usage_sources`, and computed `updated_at`. Import by existing ID. Unknown server fields fail read rather than being dropped during an update.

```terraform
resource "grafana-adaptive-metrics_recommendation_configuration" "global" {
  id            = "global-default"
  name          = "global-default"
  usage_sources = ["dashboard", "rules", "queries"]
  lifecycle { prevent_destroy = true }
}

import {
  to = grafana-adaptive-metrics_recommendation_configuration.global
  id = "global-default"
}
```

Use the exact live name and source list during adoption; the example is not a default configuration to apply blindly. Tenant and segment lifecycles create the underlying configuration objects. Create manages settings of an already-existing ID and refuses a missing parent. Destroy relinquishes Terraform settings management and preserves the parent-owned object, with an explicit warning; use `prevent_destroy` to retain ownership. This resource never creates/deletes segments or generated aggregation rules and never changes auto-apply settings.
