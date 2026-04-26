# CLAUDE.md – Terraform Provider für Serverguard24

## Projektübersicht

Terraform Provider für [Serverguard24](https://www.serverguard24.de), einem deutschen Server-Monitoring-Dienst.
Der Provider ermöglicht es, Server, Monitoring-Checks, Kontakte und Wartungsfenster vollständig als
Infrastructure-as-Code zu verwalten.

**Status:** Funktional komplett (v0.1.0) – alle geplanten Resources und Data Sources sind implementiert.

---

## Technologie-Stack

- **Sprache:** Go 1.21+
- **Framework:** [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) v1.4.2
- **API:** Serverguard24 JSON API (POST-basiert, kein REST)
- **Registry-Namespace:** `registry.terraform.io/dg-i/serverguard24`
- **Referenz:** [HashiCorp Scaffolding Template](https://github.com/hashicorp/terraform-provider-scaffolding-framework)

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
**Fehlercodes:**
- `100` – Erfolg
- `200` – Unbekannter Befehl
- `210` – Ungültiger Parameter (wird auch als "not found" behandelt)
- `220` – Authentifizierung fehlgeschlagen
- `230` – Remote-IP ungültig
- `270` – Unzureichende API-Rechte
- `310` – Ressource existiert bereits

**Client-Hilfsfunktionen:** `IsNotFound(err)` (210), `IsPermissionError(err)` (220/270), `IsAlreadyExists(err)` (310)

---

## Resources & Data Sources

### Resources (vollständiger CRUD + Import)

| Resource | API-Commands | Import-Format |
|---|---|---|
| `serverguard24_host` | `host_add`, `host_edit`, `host_delete`, `get_hosts` | Host-ID |
| `serverguard24_contact` | `contact_add`, `contact_edit`, `contact_delete`, `get_contacts` | Contact-ID |
| `serverguard24_check` | `service_add`, `service_edit`, `service_delete`, `service_show` | Check-ID |
| `serverguard24_check_contact_link` | `service_link_contact`, `service_unlink_contact` | `check_id/contact_id` |
| `serverguard24_check_distributor_link` | `service_link_distributor`, `service_unlink_distributor` | `check_id/distributor_id` |
| `serverguard24_check_integration_link` | `service_link_integration`, `service_unlink_integration` | `check_id/integration_id` |
| `serverguard24_downtime` | `downtime_add`, `downtime_link`, `downtime_unlink` (kein Edit/Delete) | Downtime-ID |

### Data Sources (nur Lesen)

| Data Source | API-Command | Hinweis |
|---|---|---|
| `serverguard24_distributors` | `get_distributors` | Leere Liste bei Berechtigungsfehler |
| `serverguard24_groups` | `get_groups` | Leere Liste bei Berechtigungsfehler |
| `serverguard24_integrations` | `get_integrations` | Leere Liste bei Berechtigungsfehler |
| `serverguard24_nodes` | `get_nodes` | Leere Liste bei Berechtigungsfehler |

Data Sources geben bei Berechtigungsfehlern (220/270) eine leere Liste zurück, statt einen Fehler auszulösen.

---

## Resource: serverguard24_host

**Attribute:**
- `id` (computed)
- `name` (required)
- `address` (required)
- `groups` (optional, list of int64)

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

Komplexeste Resource. Der Check-Typ wird über das Attribut `type` als **String-Name** gesteuert
(z.B. `"http"`, `"ssl_cert"`, `"ping"`). Die Zuordnung zu API-Integer-IDs erfolgt intern.

**Gemeinsame Attribute:**
- `id` (computed)
- `host_id` (required)
- `type` (required, String – siehe Check-Typen)
- `shortname` (optional, max 40 Zeichen)
- `description` (optional)
- `address` (optional)
- `region` (optional: `europe`, `americas`, `europe_americas`, `asia_pacific`, `world`; numeric IDs 1–5 also accepted)
- `ipv6` (optional, bool)
- `interval` (optional, float, Minuten)
- `notify_after` (optional, float, Minuten)
- `reminder_interval` (optional, float, Stunden)
- `message` (optional, text)
- `active` (optional, bool – Check aktivieren/deaktivieren)
- `notification` (optional, bool)
- Benachrichtigungsfelder: `email_critical`, `email_warning`, `email_ok`, `email_remind`,
  `sms_critical`, `sms_warning`, `sms_ok`, `sms_remind`,
  `push_critical`, `push_warning`, `push_ok`, `push_remind`,
  `http_critical`, `http_warning`, `http_ok`, `http_remind`,
  `phone_critical`, `phone_warning`, `phone_ok`, `phone_remind` (alle optional, bool)

### Check-Typen und typ-spezifische Parameter

| Typ-Name | API-ID | Pflichtfelder (zusätzlich) |
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
| `blacklist` | 54 | (diverse blacklist_* booleans) |
| `mysql` | 70 | `port`, `username`, `password`, `timeout` |
| `postgresql` | 72 | `port`, `username`, `password`, `database`, `timeout` |
| `mssql` | 73 | `username`, `password`, `sql_query`, `timeout`, `port` |
| `mongodb` | 74 | `connection`, `port`, `database`, `username`, `password`, `timeout` |
| `local_web` | 110 | `path`, `searchitem_ok`, `searchitem_critical`, `timeout`, `port`, `searchitem_none` |
| `local` | 100 | `port`, `command`, `timeout` |
| `intranet` | 101 | `command` |
| `cron` | 102 | `cron`, `timezone`, `timeout` |
| `system` | 103 | `path`, `token`, `port`, `timeout`, `searchitem_none` |
| `transact_server` | 120 | `transaction_id` |
| `transact_browser` | 121 | `transaction_id` |
| `selenium` | 122 | `definition`, `language`, `timeout` |
| `postman` | 123 | `definition`, `timeout` |
| `port_scan` | 130 | `port_range`, `portscan_none` |

---

## Resource: serverguard24_downtime

**Hinweis:** Kein `edit` oder `delete` in der API – alle Attribute verwenden `ForceNew` (Destroy + Recreate).

**Attribute:**
- `id` (computed)
- `hostname` (required)
- `start_time` (required, RFC 3339 Format, z.B. `"2026-04-01T00:00:00Z"` – wird intern zu Unix-Timestamp konvertiert)
- `end_time` (required, RFC 3339 Format)
- `description` (required)
- `type` (optional, default `"maintenance"`)
- `readonly` (optional, bool)
- `services` (optional, list of int – via `downtime_link`)
- `hosts` (optional, list of int – via `downtime_link`)

---

## Projektstruktur

```
terraform-provider-serverguard24/
├── .github/
│   └── workflows/
│       ├── test.yml                # CI: Build + Unit-Tests + Linting
│       └── release.yml             # CD: GPG-signiertes Release via GoReleaser
├── api-docs/                       # Serverguard24 API-Referenzdokumentation
├── docs/                           # Terraform Registry Dokumentation
│   ├── index.md
│   ├── resources/                  # 7 Resource-Docs
│   └── data-sources/               # 4 Data-Source-Docs
├── examples/
│   ├── basic/main.tf               # Vollständiges Arbeitsbeispiel
│   ├── provider/provider.tf
│   ├── resources/                   # Einzelne Resource-Beispiele
│   └── data-sources/                # Einzelne Data-Source-Beispiele
├── internal/
│   ├── provider/
│   │   └── provider.go             # Provider-Definition, Konfiguration, Auth
│   ├── client/
│   │   ├── client.go               # API-Client (HTTP, Token, Fehlerklassifikation)
│   │   └── client_test.go
│   ├── resources/
│   │   ├── host.go
│   │   ├── contact.go
│   │   ├── check.go                # Komplexeste Resource (28 Check-Typen)
│   │   ├── downtime.go
│   │   ├── check_link.go           # Generische Link-Implementierung (3 Typen)
│   │   ├── check_test.go
│   │   └── check_link_test.go
│   └── datasources/
│       ├── groups.go
│       ├── distributors.go
│       ├── integrations.go
│       └── nodes.go
├── main.go
├── GNUmakefile                     # build, install, dev, test, testacc, lint, generate
├── .goreleaser.yml                 # Multi-Plattform-Build + GPG-Signierung
├── .golangci.yml
├── go.mod / go.sum
├── terraform-registry-manifest.json
├── CHANGELOG.md
├── LICENSE
└── CLAUDE.md
```

---

## Build & Entwicklung

```bash
make build      # Binary bauen
make install    # In ~/.terraform.d/plugins installieren
make dev        # Lokale Installation mit dev_overrides (~/.terraformrc)
make test       # Unit-Tests (120s Timeout)
make testacc    # Acceptance-Tests (TF_ACC=1)
make lint       # golangci-lint
make generate   # Dokumentation generieren (tfplugindocs)
make tidy       # go mod tidy
```

**Lokale Entwicklung:** `make dev` schreibt eine `~/.terraformrc` mit `dev_overrides`, sodass
`terraform plan`/`tofu plan` direkt ohne `init` mit dem lokalen Build funktioniert.

**Nach jeder Interface-Änderung** (neue/umbenannte Attribute, neue Resources/Data Sources, geänderte Typen):
1. `make generate` – regeneriert `docs/` aus Schema-Beschreibungen (tfplugindocs)
2. `make dev` – installiert den neuen Build lokal
3. README.md prüfen – enthält manuelle Beispiele, die nicht automatisch aktualisiert werden

---

## CI/CD

- **test.yml:** Läuft bei Push/PR – Build, Unit-Tests, Linting (golangci-lint)
- **release.yml:** Ausgelöst durch `v*`-Tags – GPG-signiertes Release via GoReleaser
- **GoReleaser:** Multi-Plattform (FreeBSD, Windows, Linux, Darwin), Multi-Architektur (amd64, arm64, etc.), ZIP-Archive mit SHA256SUMS

---

## Provider-Konfiguration

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
  api_token = var.serverguard24_token  # alternativ: SG24_API_TOKEN env var
}
```

Der `api_token` kann auch über die Umgebungsvariable `SG24_API_TOKEN` gesetzt werden.
Die explizite Konfiguration hat Vorrang vor der Umgebungsvariable.

---

## Beispiel-Nutzung

```hcl
resource "serverguard24_host" "webserver" {
  name    = "My Webserver"
  address = "192.168.1.100"
}

resource "serverguard24_check" "http" {
  host_id   = serverguard24_host.webserver.id
  type      = "http"
  shortname = "HTTP Check"
  path      = "/"
  timeout   = 10
  port      = 443
  active    = false

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

resource "serverguard24_downtime" "maintenance" {
  hostname    = serverguard24_host.webserver.name
  start_time  = "2026-04-01T00:00:00Z"
  end_time    = "2026-04-01T02:00:00Z"
  description = "Planned maintenance window"
  services    = [tonumber(serverguard24_check.http.id)]
}
```

---

## Bekannte Einschränkungen der API

1. **Keine REST-Semantik** – alle Operationen gehen über einen einzigen POST-Endpunkt
2. **Wartungsfenster** – kein `downtime_edit` und kein `downtime_delete` → `ForceNew` in Terraform
3. **Verteilerlisten/Gruppen/Integrationen** – kein Create/Edit/Delete → nur als Data Sources
4. **Fehlercode 210** – dient doppelt als "Parameter ungültig" und "Ressource nicht gefunden"; der Client behandelt es konservativ als "gone"
5. **Import** – möglich für alle Resources, erfordert aber manuelle ID-Ermittlung im Serverguard24-Dashboard
6. **Batch-Requests** – die API unterstützt mehrere Commands pro Request; aktuell wird ein Command pro Request verwendet (Optimierung später möglich)

---

## Offene Punkte / TODOs

- [x] Acceptance Tests gegen die echte API (mit Test-Account) — `internal/acceptance/`, läuft via `make testacc` und in GitHub Actions (job `testacc`, benötigt Secret `SG24_API_TOKEN`)
- [ ] Bei Serverguard24 nachfragen: `downtime_delete`, `distributor_add/edit/delete`
- [ ] **Groups-Feature: Status unklar** — `get_groups` liefert immer eine leere Liste, kein UI-Zugang, keine API-Commands zum Anlegen/Bearbeiten von Gruppen; `groups`-Attribut am Host und `serverguard24_groups` Data Source als "Status unbekannt" markiert
- [ ] Veröffentlichung auf registry.terraform.io — GPG-Signing ist vorbereitet (`.goreleaser.yml`); benötigt noch: GPG-Schlüsselpaar generieren, `GPG_PRIVATE_KEY` + `PASSPHRASE` als GitHub-Secrets setzen, Public Key bei registry.terraform.io registrieren
