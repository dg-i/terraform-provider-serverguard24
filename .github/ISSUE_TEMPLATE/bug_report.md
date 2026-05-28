---
name: Bug report
about: Report something that doesn't work as documented
title: ""
labels: bug
assignees: ""
---

<!--
Thanks for taking the time to file a report. Please complete the sections
below so we can reproduce the problem quickly. If a section doesn't apply,
write "n/a" rather than deleting it.

For security issues, do NOT file a public issue — see SECURITY.md.
-->

## Versions

- Provider version: <!-- e.g. 0.1.2 -->
- Terraform / OpenTofu version: <!-- output of `terraform version` or `tofu version` -->
- Go version (only if building from source): <!-- output of `go version` -->
- OS / arch: <!-- e.g. linux/amd64, darwin/arm64 -->

## What happened

<!-- A clear, concise description of the actual behavior. -->

## What you expected to happen

<!-- A clear, concise description of the expected behavior. -->

## Minimal Terraform configuration to reproduce

```hcl
# Replace with the smallest config that reproduces the issue.
# Redact any tokens, hostnames, or other sensitive values.
```

## Steps to reproduce

1. `terraform init`
2. `terraform plan`
3. ...

## Full error output

<details>
<summary>Output (with <code>TF_LOG=DEBUG</code> if available)</summary>

```
# paste the full output here — redact API tokens and personal data
```

</details>

## Additional context

<!--
Anything else that might help: recent changes, workarounds you've tried,
related issues, screenshots of the Serverguard24 dashboard, etc.
-->
