---
page_title: "serverguard24_check_contact_link Resource - serverguard24"
subcategory: ""
description: |-
  Links a monitoring check to a contact.
---

# serverguard24_check_contact_link

Links a monitoring check to a contact so that the contact receives notifications for that check.

All attributes are immutable — any change destroys and recreates the link.

## Example Usage

```hcl
resource "serverguard24_check_contact_link" "http_ops" {
  check_id   = serverguard24_check.http.id
  contact_id = serverguard24_contact.ops.id
}
```

## Schema

### Required

- `check_id` (String) — ID of the check. Forces replacement on change.
- `contact_id` (String) — ID of the contact. Forces replacement on change.

### Read-Only

- `id` (String) — Composite ID in the form `<check_id>/<contact_id>`.

## Import

```shell
terraform import serverguard24_check_contact_link.http_ops 12345/67890
```
