---
page_title: "serverguard24_contact Resource - serverguard24"
subcategory: ""
description: |-
  Manages a Serverguard24 contact.
---

# serverguard24_contact

Manages a Serverguard24 contact. Contacts receive alert notifications when checks change state. Link contacts to checks with [`serverguard24_check_contact_link`](check_contact_link.md).

## Example Usage

```hcl
resource "serverguard24_contact" "ops" {
  name     = "Ops Team"
  language = "en"
  email    = "ops@example.com"
  mobile   = "+49123456789"
}
```

## Schema

### Required

- `name` (String) — Display name for the contact.
- `language` (String) — Notification language. Valid values: `en`, `de`.

### Optional

- `email` (String) — Email address for alert notifications.
- `mobile` (String) — Mobile phone number for SMS alerts.
- `network` (String) — Network push address.
- `standard` (Boolean) — Whether this is the default/standard contact.

### Read-Only

- `id` (String) — Contact ID assigned by Serverguard24.

## Import

Contacts can be imported using their numeric ID from the Serverguard24 dashboard:

```shell
terraform import serverguard24_contact.ops 12345
```
