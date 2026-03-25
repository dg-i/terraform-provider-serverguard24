---
page_title: "Serverguard24 Provider"
subcategory: ""
description: |-
  The Serverguard24 provider manages monitoring resources in the Serverguard24 service.
---

# Serverguard24 Provider

The Serverguard24 provider manages resources in [Serverguard24](https://www.serverguard24.de), a German server monitoring service. It allows you to manage hosts, checks, contacts, maintenance windows, and their relationships as Infrastructure-as-Code.

## Authentication

An API token is required. It can be provided via:

1. The `api_token` provider argument
2. The `SG24_API_TOKEN` environment variable (recommended for CI/CD)

```shell
export SG24_API_TOKEN="your-api-token"
```

## Example Usage

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
  # api_token = "..." # or set SG24_API_TOKEN environment variable
}

resource "serverguard24_host" "web" {
  name    = "web.example.com"
  address = "203.0.113.10"
}

resource "serverguard24_check" "http" {
  host_id   = serverguard24_host.web.id
  type      = "http"
  shortname = "HTTP check"
  path      = "/"
  port      = 443
  timeout   = 10
}
```

## Schema

### Optional

- `api_token` (String, Sensitive) — Serverguard24 API token. Can also be set via the `SG24_API_TOKEN` environment variable.
