---
page_title: "serverguard24_integrations Data Source - serverguard24"
subcategory: ""
description: |-
  Retrieves all available Serverguard24 integrations.
---

# serverguard24_integrations

Retrieves all available Serverguard24 integrations. Use the returned IDs with `serverguard24_check_integration_link`.

## Example Usage

```hcl
data "serverguard24_integrations" "all" {}

output "integrations" {
  value = data.serverguard24_integrations.all.integrations
}

resource "serverguard24_check_integration_link" "http_jira" {
  check_id       = serverguard24_check.http.id
  integration_id = data.serverguard24_integrations.all.integrations[0].id
}
```

## Schema

### Read-Only

- `integrations` (List of Object) — List of all integrations.
  - `id` (String) — Integration ID.
  - `name` (String) — Integration display name.
