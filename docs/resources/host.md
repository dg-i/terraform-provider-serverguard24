---
page_title: "serverguard24_host Resource - serverguard24"
subcategory: ""
description: |-
  Manages a Serverguard24 host.
---

# serverguard24_host

Manages a Serverguard24 host. Hosts are the top-level objects that group monitoring checks together and represent a monitored server or device.

## Example Usage

```hcl
resource "serverguard24_host" "web" {
  name    = "web.example.com"
  address = "203.0.113.10"
}

# With group assignment
data "serverguard24_groups" "all" {}

resource "serverguard24_host" "db" {
  name    = "db.example.com"
  address = "203.0.113.20"
  groups  = [tonumber(data.serverguard24_groups.all.groups[0].id)]
}
```

## Schema

### Required

- `name` (String) — Hostname or display name shown in the Serverguard24 dashboard.
- `address` (String) — IP address or FQDN of the host.

### Optional

- `groups` (List of Number) — List of group IDs to assign this host to. Use the `serverguard24_groups` data source to look up IDs.

### Read-Only

- `id` (String) — Host ID assigned by Serverguard24.

## Import

Hosts can be imported using their numeric ID from the Serverguard24 dashboard:

```shell
terraform import serverguard24_host.web 12345
```

After importing, run `terraform plan` to verify the state matches your configuration.
