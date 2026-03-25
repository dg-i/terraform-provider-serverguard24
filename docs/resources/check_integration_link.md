---
page_title: "serverguard24_check_integration_link Resource - serverguard24"
subcategory: ""
description: |-
  Links a monitoring check to an integration.
---

# serverguard24_check_integration_link

Links a monitoring check to an integration (e.g. Jira Service Management, PagerDuty webhook). Use the `serverguard24_integrations` data source to look up integration IDs.

All attributes are immutable — any change destroys and recreates the link.

## Example Usage

```hcl
data "serverguard24_integrations" "all" {}

resource "serverguard24_check_integration_link" "http_jira" {
  check_id       = serverguard24_check.http.id
  integration_id = data.serverguard24_integrations.all.integrations[0].id
}
```

## Schema

### Required

- `check_id` (String) — ID of the check. Forces replacement on change.
- `integration_id` (String) — ID of the integration. Forces replacement on change.

### Read-Only

- `id` (String) — Composite ID in the form `<check_id>/<integration_id>`.

## Import

```shell
terraform import serverguard24_check_integration_link.http_jira 12345/67890
```
