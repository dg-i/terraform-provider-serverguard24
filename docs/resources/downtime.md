---
page_title: "serverguard24_downtime Resource - serverguard24"
subcategory: ""
description: |-
  Manages a Serverguard24 maintenance window (downtime).
---

# serverguard24_downtime

Manages a Serverguard24 maintenance window (downtime). Downtimes suppress alerts for the associated checks during a scheduled window.

~> **API limitation:** The Serverguard24 API does not support editing or deleting downtimes. Any change to a required attribute forces the resource to be destroyed and recreated. The old downtime record remains in Serverguard24 (only its service/host links are removed), but it will expire naturally.

## Example Usage

```hcl
resource "serverguard24_downtime" "maintenance" {
  hostname    = serverguard24_host.web.name
  start_time  = "2026-04-01T00:00:00Z"
  end_time    = "2026-04-01T02:00:00Z"
  description = "Scheduled maintenance"

  services = [tonumber(serverguard24_check.http.id)]
}
```

## Schema

### Required

- `hostname` (String) — Name of the host this downtime applies to. Must match the host's `name` attribute exactly.
- `start_time` (String) — Start of the maintenance window in RFC 3339 format (e.g. `2026-04-01T00:00:00Z`).
- `end_time` (String) — End of the maintenance window in RFC 3339 format. Maximum duration is 23:59:59.
- `description` (String) — Human-readable description. Used to identify the downtime if it already exists (error code 310).

### Optional

- `type` (String) — Maintenance window type. Currently only `"maintenance"` is supported. Default: `"maintenance"`.
- `readonly` (Boolean) — Mark the downtime as read-only in the Serverguard24 UI.
- `services` (List of Number) — Check IDs to suppress alerts for during this window.
- `hosts` (List of Number) — Host IDs to suppress alerts for during this window.

### Read-Only

- `id` (String) — Downtime ID assigned by Serverguard24.

## Import

Downtimes can be imported using their numeric ID. Because the API has no read endpoint for downtimes, only `id` is populated after import. Add all other attributes to your configuration and run `terraform apply` to reconcile state.

```shell
terraform import serverguard24_downtime.maintenance 12345
```
