# AGENTS.md

Provider for Serverguard24 (<https://www.serverguard24.de>), German server monitoring service. JSON-POST API, no public docs reachable to agents — this file IS the API reference. Status: v0.2.0, functionally complete.

## Stack

- Go: see `go.mod` (currently `go 1.26.0` baseline, `go1.26.10` toolchain).
- Framework: `terraform-plugin-framework` (see `go.mod` for pinned version).
- Registry: `registry.terraform.io/dg-i/serverguard24`.

## API

- Endpoint: `POST https://www.serverguard24.de/de/api/json.html`.
- Auth: `token` field in JSON body. No HTTP header.
- Request: `{"token": "...", "requests": [{"command": "...", "k": "v"}]}`.
- Response: `[{"code": "100|...", "errmessage": "...", "data": {...}}]`.

Error codes (`internal/client/client.go`):

| Code | Meaning | Helper |
|---|---|---|
| 100 | Success | — |
| 200 | Unknown command | — |
| 210 | Invalid param / not found (overloaded) | `IsNotFound` |
| 220 | Auth failed | `IsPermissionError` |
| 230 | Remote IP invalid | — |
| 270 | Insufficient API permissions | `IsPermissionError` |
| 300 | Resource gone (undocumented; `service_show` only) | `IsNotFound` |
| 310 | Already exists | `IsAlreadyExists` |

Data sources swallow 220/270 and return `[]` instead of erroring.

## Resources

| Resource | Commands | Import format |
|---|---|---|
| `serverguard24_host` | `host_add`, `host_edit`, `host_delete`, `get_hosts` | host id |
| `serverguard24_contact` | `contact_add`, `contact_edit`, `contact_delete`, `get_contacts` | contact id |
| `serverguard24_check` | `service_add`, `service_edit`, `service_delete`, `service_show`, `service_monitoring`, `service_notification` | check id |
| `serverguard24_check_contact_link` | `service_link_contact`, `service_unlink_contact` | `check_id/contact_id` |
| `serverguard24_check_distributor_link` | `service_link_distributor`, `service_unlink_distributor` | `check_id/distributor_id` |
| `serverguard24_check_integration_link` | `service_link_integration`, `service_unlink_integration` | `check_id/integration_id` |
| `serverguard24_downtime` | `downtime_add`, `downtime_link`, `downtime_unlink` (no edit/delete) | downtime id |

## Data sources

| Data source | Command | Lookup |
|---|---|---|
| `serverguard24_host` | `get_hosts` | `id` or `name` |
| `serverguard24_contact` | `get_contacts` | `id` or `name` |
| `serverguard24_check` | `service_show` | `id` only (no bulk-list API) |
| `serverguard24_distributors` | `get_distributors` | enumerate |
| `serverguard24_groups` | `get_groups` | enumerate |
| `serverguard24_integrations` | `get_integrations` | enumerate |
| `serverguard24_nodes` | `get_nodes` | enumerate |

## Check types

`type` is a string name; mapping to API ints lives in `check_resource.go`. Type-specific params go in `params {}`.

| type | API id | params (required-extra) |
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
| `blacklist` | 54 | (`blacklist_*` bools) |
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

## Layout

```
internal/
  client/         # HTTP client, typed APIError, retry, ctx
  provider/       # all resources + data sources, one package
    check_resource.go            # most complex resource
    check_resource_fields.go     # modelSpecs / paramSpecs manifests (encode/decode)
    check_resource_snapshot_test.go + testdata/  # golden tests
    clear_warnings.go            # plan-time warnings for non-clearable fields
    *_resource.go / *_data_source.go
  acceptance/     # TestAcc* (TF_ACC=1), sweepers, CheckDestroy
docs/             # generated by tfplugindocs (`make generate`)
examples/         # registry-rendered HCL examples
api-docs/         # vendor API reference (German) — local only, gitignored, not redistributed
```

## Workflow

See `CONTRIBUTING.md` for build/test/lint commands. Additional rules:

- After any schema change: `make generate` AND check `README.md` (manual examples not regenerated). Auto-memory `feedback_docs_after_interface_changes` enforces this.
- Acceptance tests need `SG24_API_TOKEN` and create real resources; gated by `TF_ACC=1`. Sweepers via `-sweep=all`.
- Commit style: conventional prefix (`fix:`, `feat:`, `docs:`, `chore:`); see `git log` for tone.

## Gotchas

Things that have actually bitten implementers. Ignore at your peril.

### Plan modifiers: null-in-plan can't become concrete in state

`check_resource.go` defines custom `useStateXxxModifier` (String/Bool/Int64/Float64/Object) that resolves Unknown→null when state is null, unlike stock `UseStateForUnknown()`. Purpose: after `terraform import`, fields the API doesn't return stay null in plans instead of "known after apply".

Consequence: on Create, Optional+Computed fields the user didn't set are **null in the plan**. If Create then populates them from an API read-back, Terraform errors with "produced inconsistent result after apply: .X was null, but now …". This blocks the obvious "refresh after Create" pattern; an attempt was tried and reverted (see closed PRs / `git reflog`).

If you need `ImportStateVerify`, the trade-off is between this modifier and the post-create read-back — you cannot have both with the current schema.

### `service_show` does not return `host_id`

Import recovers everything except `host_id`. `applyCheckData` preserves the current `state.HostID` when `host_id` is absent from the API response (see `decodeString` fallback in `check_resource_fields.go`). Any `ImportStateVerify` step must `ImportStateVerifyIgnore: []string{"host_id"}`.

### `active` / `notification` are side-channel writes

Set via separate `service_monitoring` / `service_notification` calls, not `service_add`/`service_edit`. `service_show` may be eventually consistent for them. Update only re-issues these calls when the plan value differs from state (`check_resource.go`).

### API silently ignores "clearing" operations

The API does not accept `false`/`""` to disable a previously-enabled flag or clear a previously-set string. `ModifyPlan` emits warnings via `warnIfClearingBool` / `warnIfClearingString` / `warnIfClearingParams` (`clear_warnings.go`). Driver for `paramSpecs` carrying clearing-warning behavior — when adding a new param, choose `paramStr` / `paramBool` / `paramEnum` appropriately so warnings fire automatically.

### Adding a check param: touch exactly the manifest

`check_resource_fields.go` carries `modelSpecs` (top-level check fields) and `paramSpecs` (`params {}` block). Add one entry per new attribute; encode/decode/validation/clearing-warnings all derive from it. Enum-mapped params (`ssl_version`, `auth_type`, `performance_none`, `pop_type`, `delete_mode`, `mongo_connection`, `searchitem_none`, `portscan_none`) still need inline logic — see comments in `applyCheckData` / `buildCheckCommand`. Snapshot test `TestApplyCheckData_KitchenSink` (golden files under `testdata/`) guards encode/decode drift; regenerate with `-update-golden`.

### Error code 210 is overloaded

Means both "invalid parameter" and "not found". `IsNotFound` conservatively treats 210 as gone. `service_show` additionally returns the undocumented code 300 for deleted resources; both map to `IsNotFound`.

### Cached list endpoints

`get_hosts` / `get_contacts` return the full account inventory (no server-side id filter). The client wraps them in a 30s mutex-serialized cache; host/contact CRUD invalidates explicitly. Don't add per-record API calls assuming server-side filtering — there is none.

### Downtime has no edit/delete

All attributes `ForceNew`. Adoption on 310 ("already exists") matches `(description, hostname, starttime)` to disambiguate — do not regress to first-by-description match (`findDowntime` in `downtime_resource.go`).

## References

| What | Where |
|---|---|
| Build / test / lint commands | `CONTRIBUTING.md` |
| User-facing usage | `README.md`, `docs/`, `examples/` |
| API reference (vendor PDFs) | `api-docs/` (local only; not in the repo) |
| Release notes | `CHANGELOG.md` |
| Open-source policies | `SECURITY.md`, `CODE_OF_CONDUCT.md` |
| Prior review threads | closed PRs on GitHub |
