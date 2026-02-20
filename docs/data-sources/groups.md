# serverguard24_groups

Retrieves all available Serverguard24 host groups.

## Example usage

```hcl
data "serverguard24_groups" "all" {}

output "group_names" {
  value = { for g in data.serverguard24_groups.all.groups : g.id => g.name }
}
```

## Schema

| Name     | Type        | Description        |
| -------- | ----------- | ------------------ |
| `groups` | list(object)| List of groups.    |
| `groups[].id`   | string | Group ID.   |
| `groups[].name` | string | Group name. |
