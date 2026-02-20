# Provider documentation

Terraform Registry–style documentation lives here:

- `resources/` – one `.md` file per resource (e.g. `host.md`, `contact.md`)
- `data-sources/` – one `.md` file per data source (e.g. `groups.md`)

To generate or update docs from the provider schema, use [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs) and point the output at this directory.
