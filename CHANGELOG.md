# Changelog

## 0.2.1 (2026-10-27)

BUG FIXES:

* Release: include `terraform-registry-manifest.json` in `SHA256SUMS` so the Terraform Registry accepts the release (previous releases failed with "missing SHA256 checksum for …_manifest.json").

## 0.2.0 (2026-10-21)

FEATURES:

* **New Data Source:** `serverguard24_host` — look up an existing host by `id` or `name`.
* **New Data Source:** `serverguard24_contact` — look up an existing contact by `id` or `name`.
* **New Data Source:** `serverguard24_check` — look up an existing check by `id` (no bulk-list API exists, so name lookup is not supported).

ENHANCEMENTS:

* Provider now emits a plan-time warning when a config change attempts to clear an optional string field or disable a notification/bool flag that was previously set — the Serverguard24 API silently ignores both operations on edit, and the warning explains the workaround (`terraform taint` / dashboard).
* Provider documentation now lists the known API asymmetries (no-op clears, missing `downtime_edit`/`downtime_delete`, overloaded error code 210, O(N×M) reads) so users hit them in docs rather than at apply time.
* API errors are now exposed as a typed `*client.APIError` so error code classification (`IsNotFound`, `IsPermissionError`, `IsAlreadyExists`) no longer relies on substring matching.

BUG FIXES:

* Security: built with Go 1.26.10 and updated dependencies (`golang.org/x/crypto` v0.58.0, `golang.org/x/net` v0.61.0, `google.golang.org/grpc` v1.83.2) to resolve published vulnerabilities in the standard library HTTP/2 implementation, `x/crypto` and `grpc`.
* `serverguard24_check`: `Read` now correctly recognises an out-of-band delete. `service_show` returns the undocumented code `300` (not `210`) for a deleted check; `IsNotFound` now treats both as "gone" so the resource is removed from state instead of failing the plan.

INTERNAL:

* Acceptance tests now verify deletion via `CheckDestroy` and ship with sweepers (`-sweep=all`) that clean up leftover `acc-*` hosts and contacts from failed runs.
* `go.mod` baseline raised to `go 1.25.0` (matching `terraform-plugin-framework` v1.19.0) with an explicit `toolchain` directive; subsequently raised to `go 1.26.0` (required by current `golang.org/x/*` modules) with toolchain `go1.26.10`.

## 0.1.2 (2026-04-27)

FEATURES:

* **New Resource:** `serverguard24_host` — managed host/server with optional group assignment
* **New Resource:** `serverguard24_contact` — notification contact (email, SMS, push)
* **New Resource:** `serverguard24_check` — monitoring check with 29 check types (HTTP, HTTPS, ping, SSL cert, DNS, SSH, FTP, TCP, SMTP, POP3, IMAP, email round-trip, database checks, local/intranet agents, cron, transaction, Selenium, Postman, port scan, and more)
* **New Resource:** `serverguard24_downtime` — maintenance/downtime window
* **New Resource:** `serverguard24_check_contact_link` — links a contact to a check
* **New Resource:** `serverguard24_check_distributor_link` — links a distributor list to a check
* **New Resource:** `serverguard24_check_integration_link` — links an integration to a check
* **New Data Source:** `serverguard24_groups` — available host groups
* **New Data Source:** `serverguard24_distributors` — available distributor lists
* **New Data Source:** `serverguard24_integrations` — available integrations
* **New Data Source:** `serverguard24_nodes` — available monitoring nodes

ENHANCEMENTS:

* `serverguard24_check`: Type-specific parameters are grouped in a `params {}` nested block, keeping the top-level schema clean and making it explicit which attributes belong to which check type.
* `serverguard24_check`: Human-readable string values for all enum attributes — `type` (e.g. `"http"`, `"ssl_cert"`), `region` (e.g. `"europe"`, `"world"`), `ssl_version`, `auth_type`, `pop_type`, `delete_mode`, `mongo_connection`, `performance_none`, `searchitem_none`, `portscan_none`. Numeric IDs accepted for backwards compatibility and normalized to names on read.
* `serverguard24_check`: Config-time validation that only parameters valid for the selected check type are set, and that all required parameters are present — prevents opaque API 210 errors at apply time.
* `serverguard24_check`: `active` attribute to deactivate a check without deleting it (via `service_monitoring`).
* `serverguard24_check`: `notification` attribute to globally enable/disable alert notifications for a check (via `service_notification`).
* All resources support `terraform import` / `tofu import`.
* Data sources (`distributors`, `groups`, `integrations`, `nodes`) return an empty list on permission errors (API codes 220/270) instead of failing, allowing use in accounts without those features.
* Acceptance test suite covering all resources and data sources (`make testacc`, requires `SG24_API_TOKEN`).

## 0.1.0 (2026-04-26)

Initial tagged development release. Not published to the Terraform Registry —
the first public release was 0.1.2. Contained the foundational provider
scaffolding, the initial implementations of all resources and data sources,
import support, and the pre-commit / lint setup that the 0.1.2 release built
on. See the 0.1.2 entry for the user-facing surface.
