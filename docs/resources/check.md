---
page_title: "serverguard24_check Resource - serverguard24"
subcategory: ""
description: |-
  Manages a Serverguard24 monitoring check.
---

# serverguard24_check

Manages a Serverguard24 monitoring check. A check belongs to a host and defines the monitoring probe type, parameters, and notification behaviour.

The `type` attribute selects the check kind; different types require different additional attributes (see the type table below).

## Example Usage

```hcl
resource "serverguard24_host" "web" {
  name    = "web.example.com"
  address = "203.0.113.10"
}

# HTTP check
resource "serverguard24_check" "http" {
  host_id   = serverguard24_host.web.id
  type      = "http"
  shortname = "HTTP"
  path      = "/"
  port      = 443
  timeout   = 10
}

# SSL certificate check
resource "serverguard24_check" "ssl" {
  host_id           = serverguard24_host.web.id
  type              = "ssl_cert"
  shortname         = "SSL"
  protocol          = "https"
  port              = 443
  timeout           = 10
  validity_warning  = 14
  validity_critical = 7
}

# PING check (deactivated)
resource "serverguard24_check" "ping" {
  host_id           = serverguard24_host.web.id
  type              = "ping"
  shortname         = "PING"
  active            = false
  response_warning  = 200
  response_critical = 500
  timeout           = 10
}
```

## Schema

### Required

- `host_id` (String) — ID of the host this check belongs to.
- `type` (String) — Check type. See the [Check Types](#check-types) table below.

### Optional — General

- `shortname` (String) — Short display name (max 40 characters).
- `description` (String) — Longer description of the check.
- `address` (String) — Override the host address for this check.
- `region` (String) — Monitoring region: `1`=Europe, `2`=Americas, `3`=Europe+Americas, `4`=Asia/Pacific, `5`=World.
- `ipv6` (Boolean) — Use IPv6 for the probe.
- `interval` (Number) — Check interval in minutes.
- `notify_after` (Number) — Notify after this many minutes of downtime.
- `reminder_interval` (Number) — Reminder interval in hours.
- `message` (String) — Custom notification message.
- `active` (Boolean) — Set to `false` to deactivate the check without deleting it. Default: `true`.
- `notification` (Boolean) — Set to `false` to suppress alert notifications. Default: `true`.

### Optional — Notification Channels

Enable notifications per channel and state. All default to `false`.

| Attribute | Description |
|---|---|
| `email_critical` | Email on critical state |
| `email_warning` | Email on warning state |
| `email_ok` | Email on recovery |
| `email_remind` | Email reminders |
| `sms_critical` | SMS on critical |
| `sms_warning` | SMS on warning |
| `sms_ok` | SMS on recovery |
| `sms_remind` | SMS reminders |
| `push_critical` | Push on critical |
| `push_warning` | Push on warning |
| `push_ok` | Push on recovery |
| `push_remind` | Push reminders |
| `http_critical` | HTTP webhook on critical |
| `http_warning` | HTTP webhook on warning |
| `http_ok` | HTTP webhook on recovery |
| `http_remind` | HTTP webhook reminders |
| `phone_critical` | Phone call on critical |
| `phone_warning` | Phone call on warning |
| `phone_ok` | Phone call on recovery |
| `phone_remind` | Phone call reminders |

### Optional — Type-Specific Attributes

| Attribute | Type | Description |
|---|---|---|
| `path` | String | URL path (HTTP, HTTP Perf, Site Perf, Local Web) |
| `timeout` | Number | Timeout in seconds |
| `port` | Number | TCP port |
| `protocol` | String | Protocol, e.g. `https` (SSL Cert) |
| `validity_warning` | Number | Warn if cert expires within N days (SSL Cert) |
| `validity_critical` | Number | Critical if cert expires within N days (SSL Cert) |
| `expected_ip1` | String | Expected IP address (Domain) |
| `response_warning` | Number | Warning threshold in ms (PING) |
| `response_critical` | Number | Critical threshold in ms (PING) |
| `dns_name` | String | DNS name to resolve (DNS) |
| `warning_throughput` | Number | Warning throughput threshold (HTTP Perf, Site Perf) |
| `warning_connect` | Number | Warning connect-time threshold (HTTP Perf) |
| `warning_roundtrip` | Number | Warning roundtrip threshold (HTTP Perf, Site Perf) |
| `critical_throughput` | Number | Critical throughput threshold (HTTP Perf, Site Perf) |
| `critical_connect` | Number | Critical connect-time threshold (HTTP Perf) |
| `critical_roundtrip` | Number | Critical roundtrip threshold (HTTP Perf, Site Perf) |
| `smtp_port` | Number | SMTP port (Email) |
| `sender` | String | Sender address (Email) |
| `recipient` | String | Recipient address (Email) |
| `pop_type` | String | POP protocol type (Email) |
| `pop_host` | String | POP server host (Email) |
| `pop_user` | String | POP username (Email) |
| `pop_password` | String, Sensitive | POP password (Email) |
| `delete_mode` | String | Mail delete mode (Email) |
| `time_warning` | Number | Warning time threshold (Email) |
| `time_critical` | Number | Critical time threshold (Email) |
| `username` | String | Database username (MySQL, PostgreSQL, MSSQL, MongoDB) |
| `password` | String, Sensitive | Database password (MySQL, PostgreSQL, MSSQL, MongoDB) |
| `database` | String | Database name (PostgreSQL, MongoDB) |
| `sql_query` | String | SQL query (MSSQL) |
| `mongo_connection` | String | MongoDB connection string |
| `searchitem_ok` | String | OK search string (Local Web) |
| `searchitem_critical` | String | Critical search string (Local Web) |
| `searchitem_none` | Boolean | Match mode: none (Local Web, System) |
| `command` | String | Local command to execute (Local, Intranet) |
| `cron` | String | Cron expression (Cron) |
| `timezone` | String | Timezone for cron (Cron) |
| `token` | String | Authentication token (System) |
| `transaction_id` | String | Transaction ID (TransAct Server, TransAct Browser) |
| `definition` | String | Test definition (Selenium, Postman) |
| `language` | String | Scripting language (Selenium) |
| `port_range` | String | Port range to scan (Port Scan) |
| `portscan_none` | Boolean | Match mode: none (Port Scan) |

### Read-Only

- `id` (String) — Check ID assigned by Serverguard24.

## Check Types

| `type` value | Description | Key required attributes |
|---|---|---|
| `http` | HTTP/HTTPS availability | `path`, `timeout`, `port` |
| `http_perf` | HTTP performance | `path`, `timeout`, `port`, `warning_*`, `critical_*` |
| `site_perf` | Full page load performance | `path`, `timeout`, `port`, `warning_*`, `critical_*` |
| `ssl_cert` | SSL certificate expiry | `protocol`, `port`, `timeout`, `validity_warning`, `validity_critical` |
| `domain` | DNS domain check | `expected_ip1` |
| `ping` | ICMP ping | `response_warning`, `response_critical`, `timeout` |
| `ssh` | SSH availability | `port`, `timeout` |
| `ftp` | FTP availability | `port`, `timeout` |
| `tcp` | TCP port check | `port`, `timeout` |
| `dns` | DNS resolution | `dns_name`, `timeout` |
| `smtp` | SMTP availability | `port`, `timeout` |
| `pop3` | POP3 availability | `port`, `timeout` |
| `imap` | IMAP availability | `port`, `timeout` |
| `email` | End-to-end email round-trip | `smtp_port`, `sender`, `recipient`, `pop_*`, `delete_mode`, `time_*`, `timeout` |
| `blacklist` | Blacklist check | various `blacklist_*` flags |
| `mysql` | MySQL availability | `port`, `username`, `password`, `timeout` |
| `postgresql` | PostgreSQL availability | `port`, `username`, `password`, `timeout` |
| `mssql` | MS SQL Server | `username`, `password`, `sql_query`, `port`, `timeout` |
| `mongodb` | MongoDB availability | `mongo_connection`, `port`, `database`, `username`, `password`, `timeout` |
| `local_web` | Local web check | `path`, `searchitem_ok`, `searchitem_critical`, `timeout`, `port` |
| `local` | Local agent command | `port`, `command`, `timeout` |
| `intranet` | Intranet check | `command` |
| `cron` | Cron job heartbeat | `cron`, `timezone`, `timeout` |
| `system` | System metrics | `path`, `token`, `port`, `timeout` |
| `transact_server` | Transaction (server-side) | `transaction_id` |
| `transact_browser` | Transaction (browser) | `transaction_id` |
| `selenium` | Selenium test | `definition`, `language`, `timeout` |
| `postman` | Postman collection | `definition`, `timeout` |
| `port_scan` | Port scan | `port_range` |

## Import

Checks can be imported using their numeric ID from the Serverguard24 dashboard:

```shell
terraform import serverguard24_check.http 12345
```
