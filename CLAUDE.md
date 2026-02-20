# CLAUDE.md – Terraform Provider für Serverguard24
## Projektübersicht
Ziel ist die Entwicklung eines Terraform Providers für [Serverguard24](https://www.serverguard24.de),
einem deutschen Server-Monitoring-Dienst. Der Provider ermöglicht es, Server, Monitoring-Checks,
Kontakte und Wartungsfenster vollständig als Infrastructure-as-Code zu verwalten.
---
## Technologie-Stack
- **Sprache:** Go
- **Framework:** [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) (empfohlen gegenüber dem älteren SDK)
- **API:** Serverguard24 JSON API (POST-basiert, kein REST)
- **Registry:** Ziel ist Veröffentlichung auf [registry.terraform.io](https://registry.terraform.io)
---
## API-Grundlagen
**Endpunkt:** `POST https://www.serverguard24.de/de/api/json.html`
**Authentifizierung:** API-Token im JSON-Body (kein HTTP-Header-Auth)
**Request-Format:**
```json
{
  "token": "<API_TOKEN>",
  "requests": [
    { "command": "command_name", "param1": "value1" }
  ]
}
```
**Response-Format:**
```json
[
  { "code": "100", "errmessage": "Action executed successfully", "data": { ... } }
]
```
**Globale Fehlercodes:**
- `100` – Erfolg
- `200` – Unbekannter Befehl
- `210` – Ungültiger Parameter
- `220` – Authentifizierung fehlgeschlagen
- `230` – Remote-IP ungültig
- `270` – Unzureichende API-Rechte
---
## Geplante Ressourcen & Data Sources
### Resources (vollständiger CRUD möglich)
| Resource | API-Commands |
|---|---|
| `serverguard24_host` | `host_add`, `host_edit`, `host_delete`, `get_hosts` |
| `serverguard24_contact` | `contact_add`, `contact_edit`, `contact_delete`, `get_contacts` |
| `serverguard24_check` | `service_add`, `service_edit`, `service_delete`, `service_show` |
| `serverguard24_check_contact_link` | `service_link_contact`, `service_unlink_contact` |
| `serverguard24_check_distributor_link` | `service_link_distributor`, `service_unlink_distributor` |
| `serverguard24_check_integration_link` | `service_link_integration`, `service_unlink_integration` |
| `serverguard24_downtime` | `downtime_add`, `downtime_link`, `downtime_unlink` (kein Edit/Delete – Einschränkung) |
### Data Sources (nur Lesen)
| Data Source | API-Command |
|---|---|
| `serverguard24_distributors` | `get_distributors` |
| `serverguard24_groups` | `get_groups` |
| `serverguard24_integrations` | `get_integrations` |
| `serverguard24_nodes` | `get_nodes` |
---
## Resource: serverguard24_host
**Attribute:**
- `id` (computed)
- `name` (required)
- `address` (required)
- `groups` (optional, list of int)
---
## Resource: serverguard24_contact
**Attribute:**
- `id` (computed)
- `name` (required)
- `language` (required, `en` oder `de`)
- `email` (optional)
- `mobile` (optional)
- `network` (optional)
- `standard` (optional, bool)
---
## Resource: serverguard24_check
Dies ist die komplexeste Resource. Der Check-Typ wird über das Attribut `type` gesteuert.
Je nach Typ sind unterschiedliche zusätzliche Attribute erforderlich (siehe Abschnitt "Check-Typen").
**Gemeinsame Attribute:**
- `id` (computed)
- `host_id` (required)
- `type` (required, siehe Check-Typen)
- `shortname` (optional, max 40 Zeichen)
- `description` (optional)
- `address` (optional)
- `region` (optional: `1`=Europa, `2`=Amerika, `3`=Europa/Amerika, `4`=Asien/Pazifik, `5`=World)
- `ipv6` (optional, bool)
- `interval` (optional, float, Minuten)
- `notify_after` (optional, float, Minuten)
- `reminder_interval` (optional, float, Stunden)
- `message` (optional, text)
- Benachrichtigungsfelder: `email_critical`, `email_warning`, `email_ok`, `email_remind`,
  `sms_critical`, `sms_warning`, `sms_ok`, `sms_remind`,
  `push_critical`, `push_warning`, `push_ok`, `push_remind`,
  `http_critical`, `http_warning`, `http_ok`, `http_remind`,
  `phone_critical`, `phone_warning`, `phone_ok`, `phone_remind` (alle optional, bool)
### Check-Typen und typ-spezifische Parameter
| Typ-ID | Name | Pflichtfelder (zusätzlich) |
|---|---|---|
| `11` | HTTP | `path`, `timeout`, `port` |
| `13` | HTTP Perf | `path`, `warning_throughput`, `warning_connect`, `warning_roundtrip`, `critical_throughput`, `critical_connect`, `critical_roundtrip`, `timeout`, `port` |
| `15` | Site Perf | `path`, `warning_throughput`, `warning_roundtrip`, `critical_throughput`, `critical_roundtrip`, `timeout`, `port` |
| `14` | SSL Cert | `protocol`, `port`, `validity_warning`, `validity_critical`, `timeout` |
| `91` | Domain | `expected_ip1` |
| `40` | PING | `response_warning`, `response_critial`, `timeout` |
| `60` | SSH | `port`, `timeout` |
| `20` | FTP | `port`, `timeout` |
| `80` | TCP | `port`, `timeout` |
| `90` | DNS | `dns_name`, `timeout` |
| `51` | SMTP | `port`, `timeout` |
| `50` | POP3 | `port`, `timeout` |
| `52` | IMAP | `port`, `timeout` |
| `53` | EMAIL | `smtp_port`, `sender`, `recipient`, `pop_type`, `pop_host`, `pop_user`, `pop_password`, `delete_mode`, `time_warning`, `time_critial`, `timeout` |
| `54` | Blacklist | (diverse blacklist_* booleans) |
| `70` | MYSQL | `port`, `username`, `password`, `timeout` |
| `72` | PostGreSQL | `port`, `username`, `password`, `database`, `timeout` |
| `73` | MS SQL | `username`, `password`, `sql_query`, `timeout`, `port` |
| `74` | mongoDB | `connection`, `port`, `database`, `username`, `password`, `timeout` |
| `110` | Local Web | `path`, `searchitem_ok`, `searchitem_critical`, `timeout`, `port`, `searchitem_none` |
| `100` | Local | `port`, `command`, `timeout` |
| `101` | Intranet | `command` |
| `102` | Cron | `cron`, `timezone`, `timeout` |
| `103` | System | `path`, `token`, `port`, `timeout`, `searchitem_none` |
| `120` | TransAct Server | `transaction_id` |
| `121` | TransAct Browser | `transaction_id` |
| `122` | Selenium | `definition`, `language`, `timeout` |
| `123` | Postman | `definition`, `timeout` |
| `130` | Port Scan | `port_range`, `portscan_none` |
---
## Resource: serverguard24_downtime
**Hinweis:** Kein `edit` oder `delete` in der API – Änderungen erfordern Destroy + Recreate.
**Attribute:**
- `id` (computed)
- `hostname` (required)
- `starttime` (required, Unix-Timestamp)
- `endtime` (required, Unix-Timestamp)
- `description` (required)
- `type` (optional, default `1`)
- `readonly` (optional, bool)
- `services` (optional, list of int – via `downtime_link`)
- `hosts` (optional, list of int – via `downtime_link`)
---
## Projektstruktur (empfohlen)
```
terraform-provider-serverguard24/
├── internal/
│   ├── provider/
│   │   └── provider.go          # Provider-Definition, Konfiguration, Auth
│   ├── client/
│   │   └── client.go            # API-Client (HTTP, Token, Command-Dispatch)
│   ├── resources/
│   │   ├── host.go
│   │   ├── contact.go
│   │   ├── check.go
│   │   ├── downtime.go
│   │   └── check_link.go
│   └── datasources/
│       ├── groups.go
│       ├── distributors.go
│       ├── integrations.go
│       └── nodes.go
├── examples/
│   └── basic/
│       └── main.tf
├── docs/
├── main.go
├── GNUmakefile
├── go.mod
├── go.sum
└── CLAUDE.md
```
---
## Provider-Konfiguration (Terraform-Nutzer-Perspektive)
```hcl
terraform {
  required_providers {
    serverguard24 = {
      source  = "registry.terraform.io/<namespace>/serverguard24"
      version = "~> 0.1"
    }
  }
}
provider "serverguard24" {
  api_token = var.serverguard24_token  # alternativ: SG24_API_TOKEN env var
}
```
---
## Beispiel-Nutzung
```hcl
resource "serverguard24_host" "webserver" {
  name    = "My Webserver"
  address = "192.168.1.100"
}
resource "serverguard24_check" "http" {
  host_id  = serverguard24_host.webserver.id
  type     = 11
  shortname = "HTTP Check"
  address  = "https://example.com"
  path     = "/"
  timeout  = 10
  port     = 443
  email_critical = true
  email_ok       = true
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
```
---
## Bekannte Einschränkungen der API
1. **Keine REST-Semantik** – alle Operationen gehen über einen einzigen POST-Endpunkt
2. **Wartungsfenster** – kein `downtime_edit` und kein `downtime_delete` dokumentiert → `ForceNew` in Terraform nötig
3. **Verteilerlisten/Gruppen/Integrationen** – kein Create/Edit/Delete → nur als Data Sources nutzbar
4. **Import bestehender Ressourcen** – möglich über `get_hosts`, `get_contacts`, `host_get_services`, erfordert aber manuelle ID-Ermittlung im UI (Objekt-IDs müssen im Dashboard eingeblendet werden)
5. **Batch-Requests** – die API unterstützt mehrere Commands pro Request; für den Provider wird zunächst ein Command pro Request verwendet (Optimierung später möglich)
---
## Offene Punkte / TODOs
- [ ] API-Token-Handling über Umgebungsvariable (`SG24_API_TOKEN`)
- [ ] `terraform import` für alle Resources implementieren
- [ ] Acceptance Tests gegen die echte API (mit Test-Account)
- [ ] Vollständige Dokumentation im Terraform Registry-Format (`.tmpl`-Dateien)
- [ ] Bei Serverguard24 nachfragen: `downtime_delete`, `distributor_add/edit/delete`
- [ ] Evaluieren: Ein generischer `serverguard24_check`-Resource-Typ vs. separate Typen pro Check-Kategorie

