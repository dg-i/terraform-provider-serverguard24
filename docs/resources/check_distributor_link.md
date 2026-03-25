---
page_title: "serverguard24_check_distributor_link Resource - serverguard24"
subcategory: ""
description: |-
  Links a monitoring check to a distributor.
---

# serverguard24_check_distributor_link

Links a monitoring check to a distributor (mailing list). Use the `serverguard24_distributors` data source to look up distributor IDs.

All attributes are immutable — any change destroys and recreates the link.

## Example Usage

```hcl
data "serverguard24_distributors" "all" {}

resource "serverguard24_check_distributor_link" "http_dist" {
  check_id       = serverguard24_check.http.id
  distributor_id = data.serverguard24_distributors.all.distributors[0].id
}
```

## Schema

### Required

- `check_id` (String) — ID of the check. Forces replacement on change.
- `distributor_id` (String) — ID of the distributor. Forces replacement on change.

### Read-Only

- `id` (String) — Composite ID in the form `<check_id>/<distributor_id>`.

## Import

```shell
terraform import serverguard24_check_distributor_link.http_dist 12345/67890
```
