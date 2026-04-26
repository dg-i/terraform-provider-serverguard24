# Changelog

## 0.1.0 (Unreleased)

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
