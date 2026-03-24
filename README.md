# Terraform Provider for Serverguard24

This Terraform provider enables managing [Serverguard24](https://www.serverguard24.de) monitoring resources as Infrastructure-as-Code.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (for building from source)
- A Serverguard24 account with API access

## Building the Provider

```sh
git clone https://git.dg-i.net/dg-i/engineering/terraform/serverguard24-provider
cd serverguard24-provider
make build
```

## Using the Provider

```hcl
terraform {
  required_providers {
    serverguard24 = {
      source  = "registry.terraform.io/dg-i/serverguard24"
      version = "~> 0.1"
    }
  }
}

provider "serverguard24" {
  api_token = var.serverguard24_token  # or set SG24_API_TOKEN env var
}
```

## Authentication

The provider requires a Serverguard24 API token. You can provide it via:

- The `api_token` argument in the provider block
- The `SG24_API_TOKEN` environment variable

## Resources

| Resource | Description |
|---|---|
| `serverguard24_host` | A monitored host/server |
| `serverguard24_contact` | A notification contact |
| `serverguard24_check` | A monitoring check (HTTP, PING, SSL, etc.) |
| `serverguard24_check_contact_link` | Links a contact to a check |
| `serverguard24_check_distributor_link` | Links a distributor to a check |
| `serverguard24_check_integration_link` | Links an integration to a check |
| `serverguard24_downtime` | A maintenance/downtime window |

## Data Sources

| Data Source | Description |
|---|---|
| `serverguard24_groups` | Available host groups |
| `serverguard24_distributors` | Available distributor lists |
| `serverguard24_integrations` | Available integrations |
| `serverguard24_nodes` | Available monitoring nodes |

## Example Usage

```hcl
resource "serverguard24_host" "webserver" {
  name    = "My Webserver"
  address = "192.168.1.100"
}

resource "serverguard24_check" "http" {
  host_id   = serverguard24_host.webserver.id
  type      = 11
  shortname = "HTTP Check"
  address   = "https://example.com"
  path      = "/"
  timeout   = 10
  port      = 443

  email_critical = true
  email_ok       = true
}

resource "serverguard24_contact" "ops_team" {
  name     = "Ops Team"
  email    = "ops@example.com"
  language = "de"
}

resource "serverguard24_check_contact_link" "http_ops" {
  check_id   = serverguard24_check.http.id
  contact_id = serverguard24_contact.ops_team.id
}
```

See the [`examples/`](examples/) directory for more complete examples.

## Check Types

The `serverguard24_check` resource supports many check types via the `type` attribute:

| Type ID | Name | Key Additional Fields |
|---|---|---|
| `11` | HTTP | `path`, `timeout`, `port` |
| `13` | HTTP Perf | `path`, `timeout`, `port`, `warning_*`, `critical_*` |
| `14` | SSL Cert | `protocol`, `port`, `validity_warning`, `validity_critical`, `timeout` |
| `40` | PING | `response_warning`, `response_critial`, `timeout` |
| `60` | SSH | `port`, `timeout` |
| `80` | TCP | `port`, `timeout` |
| `90` | DNS | `dns_name`, `timeout` |
| `91` | Domain | `expected_ip1` |
| `51` | SMTP | `port`, `timeout` |
| `70` | MySQL | `port`, `username`, `password`, `timeout` |
| `102` | Cron | `cron`, `timezone`, `timeout` |

For a full list of check types and their parameters, see the [CLAUDE.md](CLAUDE.md) documentation.

## Known Limitations

- **Downtime windows** have no `edit` or `delete` API — changes require destroy + recreate (`ForceNew`)
- **Distributors, groups, and integrations** cannot be created or edited via API — available as data sources only
- **All API operations** use a single POST endpoint (no REST semantics)

## Development

```sh
# Build
make build

# Run tests
make test

# Install locally for testing
make install
```

## License

This provider is developed and maintained by [DG-I](https://www.dg-i.net).
