---
page_title: "serverguard24_groups Data Source - serverguard24"
subcategory: ""
description: |-
  Retrieves all available Serverguard24 host groups.
---

# serverguard24_groups

Retrieves all available Serverguard24 host groups. Use the returned IDs in the `groups` attribute of `serverguard24_host`.

## Example Usage

```hcl
data "serverguard24_groups" "all" {}

output "groups" {
  value = data.serverguard24_groups.all.groups
}

resource "serverguard24_host" "web" {
  name    = "web.example.com"
  address = "203.0.113.10"
  groups  = [tonumber(data.serverguard24_groups.all.groups[0].id)]
}
```

## Schema

### Read-Only

- `groups` (List of Object) — List of all groups.
  - `id` (String) — Group ID.
  - `name` (String) — Group display name.
