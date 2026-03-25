---
page_title: "serverguard24_distributors Data Source - serverguard24"
subcategory: ""
description: |-
  Retrieves all available Serverguard24 distributors.
---

# serverguard24_distributors

Retrieves all available Serverguard24 distributors (mailing lists). Use the returned IDs with `serverguard24_check_distributor_link`.

## Example Usage

```hcl
data "serverguard24_distributors" "all" {}

output "distributors" {
  value = data.serverguard24_distributors.all.distributors
}
```

## Schema

### Read-Only

- `distributors` (List of Object) — List of all distributors.
  - `id` (String) — Distributor ID.
  - `name` (String) — Distributor display name.
