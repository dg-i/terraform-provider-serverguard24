---
page_title: "serverguard24_nodes Data Source - serverguard24"
subcategory: ""
description: |-
  Retrieves all available Serverguard24 monitoring probe locations.
---

# serverguard24_nodes

Retrieves all available Serverguard24 monitoring nodes (probe locations). These are the distributed servers that perform the actual monitoring checks.

~> **Note:** This data source requires the `get_nodes` API permission. If the account lacks this permission, an empty list is returned with a warning rather than an error.

## Example Usage

```hcl
data "serverguard24_nodes" "all" {}

output "probe_locations" {
  value = data.serverguard24_nodes.all.nodes
}
```

## Schema

### Read-Only

- `nodes` (List of Object) — List of all monitoring nodes.
  - `ipv4` (String) — IPv4 address of the probe.
  - `ipv6` (String) — IPv6 address of the probe (may be empty).
  - `location` (String) — City or location name.
  - `country` (String) — Country name.
  - `country_code` (String) — ISO country code.
  - `region` (String) — Region name (e.g. `Europe`, `America`).
  - `region_code` (String) — Region code (e.g. `EMEA`, `NAFTA`, `APAC`).
