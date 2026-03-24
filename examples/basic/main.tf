terraform {
  required_providers {
    serverguard24 = {
      source  = "registry.terraform.io/dg-i/serverguard24"
      version = "~> 0.1"
    }
  }
}

provider "serverguard24" {
  #api_token = "REDACTED"
  api_token = "REDACTED"
}

# ---------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------

variable "integration_id" {
  description = "ID of an existing integration to link to the HTTP check. Look up IDs via the `integrations` output."
  type        = string
  default     = "60"
}

# ---------------------------------------------------------------------------
# Data Sources
# ---------------------------------------------------------------------------

data "serverguard24_groups" "all" {}

data "serverguard24_distributors" "all" {}

data "serverguard24_integrations" "all" {}

data "serverguard24_nodes" "all" {}

# ---------------------------------------------------------------------------
# Host
# ---------------------------------------------------------------------------

# EXAMPLE RESOURCE – safe to destroy at any time.
resource "serverguard24_host" "example" {
  name    = "[EXAMPLE] Terraform Provider Test Host"
  address = "192.0.2.1" # TEST-NET (RFC 5737) – not a real address
}

# ---------------------------------------------------------------------------
# Contact
# ---------------------------------------------------------------------------

# EXAMPLE RESOURCE – uses a sink address so no real notifications are sent.
resource "serverguard24_contact" "example" {
  name     = "[EXAMPLE] Terraform Provider Test Contact"
  email    = "noreply+tf-example@example.com"
  language = "de"
  standard = false
}

# ---------------------------------------------------------------------------
# Checks – monitoring disabled so no actual probes are executed
# ---------------------------------------------------------------------------

resource "serverguard24_check" "http" {
  host_id    = serverguard24_host.example.id
  type       = 11
  shortname  = "[EXAMPLE] HTTP"
  monitoring = false
  path       = "/"
  timeout    = 10
  port       = 443
}

resource "serverguard24_check" "ssl" {
  host_id    = serverguard24_host.example.id
  type       = 14
  shortname  = "[EXAMPLE] SSL Cert"
  monitoring = false
  protocol   = "https"
  port       = 443
  timeout    = 10

  validity_warning  = 14
  validity_critical = 7
}

resource "serverguard24_check" "ping" {
  host_id    = serverguard24_host.example.id
  type       = 40
  shortname  = "[EXAMPLE] PING"
  monitoring = false

  response_warning  = 200
  response_critical = 500
  timeout           = 10
}

# ---------------------------------------------------------------------------
# Check–Contact link
# ---------------------------------------------------------------------------

resource "serverguard24_check_contact_link" "http_example" {
  check_id   = serverguard24_check.http.id
  contact_id = serverguard24_contact.example.id
}

# ---------------------------------------------------------------------------
# Check–Integration link
# ---------------------------------------------------------------------------

resource "serverguard24_check_integration_link" "http_int" {
  check_id       = serverguard24_check.http.id
  integration_id = var.integration_id
}

# ---------------------------------------------------------------------------
# Downtime window
# ---------------------------------------------------------------------------

resource "serverguard24_downtime" "example" {
  hostname    = serverguard24_host.example.name
  starttime   = 1775001600 # 2026-04-01 00:00:00 UTC
  endtime     = 1775008800 # 2026-04-01 02:00:00 UTC
  description = "[EXAMPLE] Terraform provider test – safe to remove"
  type        = 1

  services = [tonumber(serverguard24_check.http.id)]
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

output "host_id" {
  value = serverguard24_host.example.id
}

output "http_check_id" {
  value = serverguard24_check.http.id
}

output "contact_id" {
  value = serverguard24_contact.example.id
}

output "groups" {
  description = "All available groups. Use an ID from this list as var.test_group_id."
  value       = data.serverguard24_groups.all.groups
}

output "distributors" {
  description = "All available distributors. Use an ID from this list as var.distributor_id."
  value       = data.serverguard24_distributors.all.distributors
}

output "integrations" {
  description = "All available integrations. Use an ID from this list as var.integration_id."
  value       = data.serverguard24_integrations.all.integrations
}

output "nodes" {
  description = "Monitoring probe locations (ipv4, ipv6, location, country, region)."
  value       = data.serverguard24_nodes.all.nodes
}
