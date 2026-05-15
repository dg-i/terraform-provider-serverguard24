# Security Policy

## Supported versions

Only the latest minor release line receives security fixes.

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1   | :x:                |

## Reporting a vulnerability

If you believe you have found a security vulnerability in this provider,
**please do not file a public GitHub issue.**

Instead, report it privately by emailing:

**security@dg-i.net**

Please include:

- A description of the vulnerability and its impact
- Steps to reproduce, including a minimal Terraform configuration if applicable
- The provider version and your Go version
- Any suggested mitigation, if you have one

We will acknowledge receipt within 5 business days and aim to provide a more
detailed response — including next steps and an expected fix timeline — within
10 business days.

## Scope

This policy covers the provider code itself. Vulnerabilities in the upstream
Serverguard24 service should be reported to Serverguard24 directly via their
support channels.

## Disclosure

Once a fix is available, we will coordinate a disclosure timeline with the
reporter. Credit will be given in the release notes unless the reporter prefers
to remain anonymous.
