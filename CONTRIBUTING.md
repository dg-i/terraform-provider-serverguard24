# Contributing to terraform-provider-serverguard24

Thanks for your interest in improving the Terraform Provider for Serverguard24.
This document covers how to set up a development environment, run tests, and
submit changes.

## Requirements

- Go (see `go.mod` for the minimum version — currently 1.25)
- Terraform >= 1.6 or OpenTofu >= 1.6 for local end-to-end testing
- A Serverguard24 API token for acceptance tests (set `SG24_API_TOKEN`)

## Development workflow

```bash
make build      # Build the provider binary
make dev        # Install locally via dev_overrides in ~/.terraformrc
make test       # Unit tests
make lint       # golangci-lint
make generate   # Regenerate docs/ from schema descriptions
```

After any schema or attribute change, run `make generate` and re-check
`README.md` (manual examples there are not regenerated).

### Local end-to-end testing

`make dev` writes a `~/.terraformrc` with `dev_overrides` pointed at your build,
so `terraform plan` / `tofu plan` work without `terraform init`.

### Acceptance tests

```bash
export SG24_API_TOKEN=<your token>
make testacc
```

Acceptance tests create real resources in the configured Serverguard24 account
and clean them up afterwards. Failed runs may leave `acc-*` resources behind;
run `TF_ACC=1 go test ./internal/acceptance/... -sweep=all` to clean them up.

## Branching and commits

- Branch from `main`. PRs target `main`.
- Keep commits focused — separate refactors, features, and fixes.
- Run `make test lint` before pushing.

## Submitting a pull request

1. Fork the repo and create a topic branch.
2. Make your changes with tests where applicable.
3. Run `make test lint` and, if you added/changed schema, `make generate`.
4. Open a PR using the template. Describe the change and link any issue.
5. CI will run build, unit tests, lint, and (for same-repo PRs) acceptance tests.

## Reporting bugs

Use the issue template. Include a minimal Terraform config that reproduces the
problem, the provider version, and the full error output.

## Security issues

See [SECURITY.md](SECURITY.md) — please do not file public issues for security
problems.

## Code of conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).
