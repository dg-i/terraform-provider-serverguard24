# CLAUDE.md – Terraform Provider for Serverguard24

## Project Overview

Terraform Provider for [Serverguard24](https://www.serverguard24.de), a German server monitoring service.
The provider enables managing servers, monitoring checks, contacts, and maintenance windows fully as
Infrastructure-as-Code.

**Status:** Functionally complete (v0.1.2) — all planned resources and data sources are implemented.

---

## Technology Stack

- **Language:** Go 1.21+
- **Framework:** [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) v1.4.2
- **API:** Serverguard24 JSON API (POST-based, not REST)
- **Registry Namespace:** `registry.terraform.io/dg-i/serverguard24`
- **Reference:** [HashiCorp Scaffolding Template](https://github.com/hashicorp/terraform-provider-scaffolding-framework)

---

## API Basics

**Endpoint:** `POST https://www.serverguard24.de/de/api/json.html`
**Authentication:** API token in the JSON body (no HTTP header auth)
**Request format:**
```json
{
  "token": "<API_TOKEN>",
  "requests": [
    { "command": "command_name", "param1": "value1" }
  ]
}
```
**Response format:**
```json
[
  { "code": "100", "errmessage": "Action executed successfully", "data": { ... } }
]
```
**Error codes:**
- `100` – Success
- `200` – Unknown command
- `210` – Invalid parameter (also used as "not found")
- `220` – Authentication failed
- `230` – Remote IP invalid
- `270` – Insufficient API permissions
- `310` – Resource already exists

**Client helper functions:** `IsNotFound(err)` (210), `IsPermissionError(err)` (220/270), `IsAlreadyExists(err)` (310)

---

## Resources & Data Sources

### Resources (full CRUD + import)

| Resource | API Commands | Import Format |
|---|---|---|
| `serverguard24_host` | `host_add`, `host_edit`, `host_delete`, `get_hosts` | Host ID |
| `serverguard24_contact` | `contact_add`, `contact_edit`, `contact_delete`, `get_contacts` | Contact ID |
| `serverguard24_check` | `service_add`, `service_edit`, `service_delete`, `service_show` | Check ID |
| `serverguard24_check_contact_link` | `service_link_contact`, `service_unlink_contact` | `check_id/contact_id` |
| `serverguard24_check_distributor_link` | `service_link_distributor`, `service_unlink_distributor` | `check_id/distributor_id` |
| `serverguard24_check_integration_link` | `service_link_integration`, `service_unlink_integration` | `check_id/integration_id` |
| `serverguard24_downtime` | `downtime_add`, `downtime_link`, `downtime_unlink` (no edit/delete) | Downtime ID |

### Data Sources (read-only)

| Data Source | API Command | Notes |
|---|---|---|
| `serverguard24_distributors` | `get_distributors` | Returns empty list on permission error |
| `serverguard24_groups` | `get_groups` | Returns empty list on permission error |
| `serverguard24_integrations` | `get_integrations` | Returns empty list on permission error |
| `serverguard24_nodes` | `get_nodes` | Returns empty list on permission error |

Data sources return an empty list on permission errors (220/270) instead of raising an error.

---

## Resource: serverguard24_host

**Attributes:**
- `id` (computed)
- `name` (required)
- `address` (required)
- `groups` (optional, list of int64)

---

## Resource: serverguard24_contact

**Attributes:**
- `id` (computed)
- `name` (required)
- `language` (required, `en` or `de`)
- `email` (optional)
- `mobile` (optional)
- `network` (optional)
- `standard` (optional, bool)

---

## Resource: serverguard24_check

Most complex resource. The check type is controlled via the `type` attribute as a **string name**
(e.g. `"http"`, `"ssl_cert"`, `"ping"`). The mapping to API integer IDs is handled internally.

Type-specific parameters go inside the `params {}` nested block.

**Common attributes:**
- `id` (computed)
- `host_id` (required)
- `type` (required, string — see check types)
- `shortname` (optional, max 40 characters)
- `description` (optional)
- `address` (optional)
- `region` (optional: `europe`, `americas`, `europe_americas`, `asia_pacific`, `world`; numeric IDs 1–5 also accepted)
- `ipv6` (optional, bool)
- `interval` (optional, float, minutes)
- `notify_after` (optional, float, minutes)
- `reminder_interval` (optional, float, hours)
- `message` (optional, text)
- `active` (optional, bool — activate/deactivate the check)
- `notification` (optional, bool)
- Notification flags: `email_critical`, `email_warning`, `email_ok`, `email_remind`,
  `sms_critical`, `sms_warning`, `sms_ok`, `sms_remind`,
  `push_critical`, `push_warning`, `push_ok`, `push_remind`,
  `http_critical`, `http_warning`, `http_ok`, `http_remind`,
  `phone_critical`, `phone_warning`, `phone_ok`, `phone_remind` (all optional, bool)

### Check Types and Type-Specific Parameters

| Type Name | API ID | Required params (additional) |
|---|---|---|
| `http` | 11 | `path`, `timeout`, `port` |
| `http_perf` | 13 | `path`, `warning_throughput`, `warning_connect`, `warning_roundtrip`, `critical_throughput`, `critical_connect`, `critical_roundtrip`, `timeout`, `port` |
| `site_perf` | 15 | `path`, `warning_throughput`, `warning_roundtrip`, `critical_throughput`, `critical_roundtrip`, `timeout`, `port` |
| `ssl_cert` | 14 | `protocol`, `port`, `validity_warning`, `validity_critical`, `timeout` |
| `domain` | 91 | `expected_ip1` |
| `ping` | 40 | `response_warning`, `response_critical`, `timeout` |
| `ssh` | 60 | `port`, `timeout` |
| `ftp` | 20 | `port`, `timeout` |
| `tcp` | 80 | `port`, `timeout` |
| `dns` | 90 | `dns_name`, `timeout` |
| `smtp` | 51 | `port`, `timeout` |
| `pop3` | 50 | `port`, `timeout` |
| `imap` | 52 | `port`, `timeout` |
| `email` | 53 | `smtp_port`, `sender`, `recipient`, `pop_type`, `pop_host`, `pop_user`, `pop_password`, `delete_mode`, `time_warning`, `time_critical`, `timeout` |
| `blacklist` | 54 | (various blacklist_* booleans) |
| `mysql` | 70 | `port`, `username`, `password`, `timeout` |
| `postgresql` | 72 | `port`, `username`, `password`, `database`, `timeout` |
| `mssql` | 73 | `username`, `password`, `sql_query`, `timeout`, `port` |
| `mongodb` | 74 | `connection`, `port`, `database`, `username`, `password`, `timeout` |
| `local_web` | 110 | `path`, `searchitem_ok`, `searchitem_critical`, `timeout`, `port`, `searchitem_none` |
| `local` | 100 | `port`, `command`, `timeout` |
| `intranet` | 101 | `command` |
| `cron` | 102 | `cron`, `timezone`, `timeout` |
| `system` | 103 | `path`, `port`, `timeout`, `searchitem_none` |
| `transact_server` | 120 | `transaction_id` |
| `transact_browser` | 121 | `transaction_id` |
| `selenium` | 122 | `definition`, `language`, `timeout` |
| `postman` | 123 | `definition`, `timeout` |
| `port_scan` | 130 | `port_range`, `portscan_none` |

---

## Resource: serverguard24_downtime

**Note:** No `edit` or `delete` in the API — all attributes use `ForceNew` (destroy + recreate).

**Attributes:**
- `id` (computed)
- `hostname` (required)
- `start_time` (required, RFC 3339 format, e.g. `"2026-04-01T00:00:00Z"` — converted internally to Unix timestamp)
- `end_time` (required, RFC 3339 format)
- `description` (required)
- `type` (optional, default `"maintenance"`)
- `readonly` (optional, bool)
- `services` (optional, list of int — via `downtime_link`)
- `hosts` (optional, list of int — via `downtime_link`)

---

## Project Structure

```
terraform-provider-serverguard24/
├── .github/
│   └── workflows/
│       ├── test.yml                # CI: Build + Unit Tests + Linting
│       └── release.yml             # CD: GPG-signed release via GoReleaser
├── api-docs/                       # Serverguard24 API reference documentation
├── docs/                           # Terraform Registry documentation
│   ├── index.md
│   ├── resources/                  # 7 resource docs
│   └── data-sources/               # 4 data source docs
├── examples/
│   ├── provider/provider.tf
│   ├── resources/                   # Individual resource examples
│   └── data-sources/                # Individual data source examples
├── internal/
│   ├── provider/
│   │   └── provider.go             # Provider definition, configuration, auth
│   ├── client/
│   │   ├── client.go               # API client (HTTP, token, error classification)
│   │   └── client_test.go
│   ├── resources/
│   │   ├── host.go
│   │   ├── contact.go
│   │   ├── check.go                # Most complex resource (29 check types)
│   │   ├── downtime.go
│   │   ├── check_link.go           # Generic link implementation (3 types)
│   │   ├── check_test.go
│   │   └── check_link_test.go
│   └── datasources/
│       ├── groups.go
│       ├── distributors.go
│       ├── integrations.go
│       └── nodes.go
├── main.go
├── GNUmakefile                     # build, install, dev, test, testacc, lint, generate
├── .goreleaser.yml                 # Multi-platform build + GPG signing
├── .golangci.yml
├── go.mod / go.sum
├── terraform-registry-manifest.json
├── CHANGELOG.md
├── LICENSE
└── CLAUDE.md
```

---

## Build & Development

```bash
make build      # Build binary
make install    # Install to ~/.terraform.d/plugins
make dev        # Local install with dev_overrides (~/.terraformrc)
make test       # Unit tests (120s timeout)
make testacc    # Acceptance tests (TF_ACC=1)
make lint       # golangci-lint
make generate   # Generate documentation (tfplugindocs)
make tidy       # go mod tidy
```

**Local development:** `make dev` writes a `~/.terraformrc` with `dev_overrides` so that
`terraform plan`/`tofu plan` works directly without `init` using the local build.

**After every interface change** (new/renamed attributes, new resources/data sources, changed types):
1. `make generate` — regenerates `docs/` from schema descriptions (tfplugindocs)
2. `make dev` — installs the new build locally
3. Check README.md — contains manual examples that are not updated automatically

---

## CI/CD

- **test.yml:** Runs on push/PR — build, unit tests, linting (golangci-lint)
- **release.yml:** Triggered by `v*` tags — GPG-signed release via GoReleaser
- **GoReleaser:** Multi-platform (FreeBSD, Windows, Linux, Darwin), multi-architecture (amd64, arm64, etc.), ZIP archives with SHA256SUMS

---

## Provider Configuration

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
  api_token = var.serverguard24_token  # alternatively: SG24_API_TOKEN env var
}
```

The `api_token` can also be set via the `SG24_API_TOKEN` environment variable.
Explicit configuration takes precedence over the environment variable.

---

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
  active    = false

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

resource "serverguard24_check_contact_link" "link" {
  check_id   = serverguard24_check.http.id
  contact_id = serverguard24_contact.ops_team.id
}

resource "serverguard24_downtime" "maintenance" {
  hostname    = serverguard24_host.webserver.name
  start_time  = "2026-04-01T00:00:00Z"
  end_time    = "2026-04-01T02:00:00Z"
  description = "Planned maintenance window"
  services    = [tonumber(serverguard24_check.http.id)]
}
```

---

## Known API Limitations

1. **No REST semantics** — all operations go through a single POST endpoint
2. **Maintenance windows** — no `downtime_edit` and no `downtime_delete` → `ForceNew` in Terraform
3. **Distributor lists/groups/integrations** — no create/edit/delete → read-only as data sources
4. **Error code 210** — serves dual purpose as "invalid parameter" and "resource not found"; the client conservatively treats it as "gone"
5. **Import** — possible for all resources, but requires manual ID lookup in the Serverguard24 dashboard
6. **Batch requests** — the API supports multiple commands per request; currently one command per request is used (optimization possible later)