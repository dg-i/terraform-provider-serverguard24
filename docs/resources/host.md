# serverguard24_host

Manages a Serverguard24 host (monitored server).

## Example usage

```hcl
resource "serverguard24_host" "webserver" {
  name    = "My Webserver"
  address = "192.168.1.100"
  groups  = [1, 2]
}
```

## Import

Import an existing host by ID:

```bash
terraform import serverguard24_host.example <host_id>
```

## Schema

| Name     | Type        | Required | Description                |
| -------- | ----------- | -------- | -------------------------- |
| `id`     | string      | No       | Host ID (computed).        |
| `name`   | string      | Yes      | Hostname or display name.  |
| `address`| string      | Yes      | IP address or FQDN.        |
| `groups` | list(number)| No       | List of group IDs.         |
