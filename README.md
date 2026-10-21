# Terraform Provider for Serverguard24

[![Test](https://github.com/dg-i/terraform-provider-serverguard24/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/dg-i/terraform-provider-serverguard24/actions/workflows/test.yml)
[![Latest release](https://img.shields.io/github/v/release/dg-i/terraform-provider-serverguard24?sort=semver)](https://github.com/dg-i/terraform-provider-serverguard24/releases)
[![Terraform Registry](https://img.shields.io/badge/registry-dg--i%2Fserverguard24-844FBA?logo=terraform&logoColor=white)](https://registry.terraform.io/providers/dg-i/serverguard24/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/dg-i/terraform-provider-serverguard24)](https://goreportcard.com/report/github.com/dg-i/terraform-provider-serverguard24)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](LICENSE)

This Terraform provider enables managing [Serverguard24](https://www.serverguard24.de) monitoring resources as Infrastructure-as-Code.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.26.2 (for building from source)
- A Serverguard24 account with API access

## Building the Provider

```sh
git clone https://github.com/dg-i/terraform-provider-serverguard24.git
cd terraform-provider-serverguard24
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
| `serverguard24_host` | Existing host (by id or name) |
| `serverguard24_contact` | Existing contact (by id or name) |
| `serverguard24_check` | Existing check (by id only) |
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
  type      = "http"
  shortname = "HTTP Check"

  email_critical = true
  email_ok       = true

  params {
    path    = "/"
    timeout = 10
    port    = 443
  }
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

## Finding IDs for import

`terraform import` / `tofu import` for `serverguard24_host` and `serverguard24_check` needs the numeric **host ID** or **check (service) ID**.

To display object IDs in the Serverguard24 web UI, enable **”Show Object IDs”** at <https://www.serverguard24.de/de/admin/support/index.html>.

## Check Types

The `serverguard24_check` resource supports many check types via the `type` attribute. Type-specific parameters go inside the `params {}` block:

| Type Name | Required `params` fields |
|---|---|
| `http` | `path`, `timeout`, `port` |
| `http_perf` | `path`, `timeout`, `port`, `warning_*`, `critical_*` |
| `ssl_cert` | `protocol`, `port`, `validity_warning`, `validity_critical`, `timeout` |
| `ping` | `response_warning`, `response_critical`, `timeout` |
| `ssh` | `port`, `timeout` |
| `tcp` | `port`, `timeout` |
| `dns` | `dns_name`, `timeout` |
| `domain` | `expected_ip1` |
| `smtp` | `port`, `timeout` |
| `mysql` | `port`, `username`, `password`, `timeout` |
| `cron` | `cron`, `timezone`, `timeout` |

For a full list of check types and their parameters, see the [resource documentation](docs/resources/check.md).

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

## Contributing

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md) for
development setup, testing, and PR guidelines. By participating, you agree
to abide by the [Code of Conduct](CODE_OF_CONDUCT.md). Security issues
should be reported privately per [SECURITY.md](SECURITY.md).

## AI-Generated Content

This provider contains AI-generated code. Parts of the source code, tests
and documentation were created with the help of AI coding assistants.

## License

Licensed under the [Mozilla Public License 2.0](LICENSE). Developed and
maintained by [DG-I](https://www.dg-i.net).
