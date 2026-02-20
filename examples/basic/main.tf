terraform {
  required_providers {
    serverguard24 = {
      source  = "registry.terraform.io/dg-i/serverguard24"
      version = "~> 0.1"
    }
  }
}

provider "serverguard24" {
  api_token = "REDACTED"

}

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

data "serverguard24_groups" "all" {}

data "serverguard24_distributors" "all" {}
