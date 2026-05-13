# Code Review — terraform-provider-serverguard24

Reviewer: Claude (Opus 4.7)
Date: 2026-05-07
Scope: full repository, with focus on OSS readiness for `github.com/dg-i/`.

---

## 1. Project structure

**Recommendation: change it.** Not because what you have is wrong, but because
it deviates from the convention every Terraform Plugin Framework user expects
from the [HashiCorp scaffolding template](https://github.com/hashicorp/terraform-provider-scaffolding-framework),
and OSS first impressions matter.

The convention is a **single `internal/provider/` package**, with files named:

```
internal/provider/
  provider.go
  host_resource.go
  host_data_source.go
  contact_resource.go
  ...
  check_link_resource.go        # the generic factory + 3 New* funcs
  client.go                     # or stays in internal/client/
```

Why this is preferable for your repo specifically (not just "convention"):

1. **Helpers are split awkwardly.** `setOptionalString`, `boolToInt`,
   `nullableString` live in `resources/contact.go`; `parsePositiveID`,
   `parseLinkID` live in `resources/check_link.go`; `parseRFC3339Unix` lives in
   `resources/downtime.go`. They're all called across files in the same package
   today, but they'd be more discoverable if the whole provider were one
   package.
2. **The `linkResource` factory pattern doesn't fit the "one resource per file"
   convention** either. With everything in `internal/provider/`, you'd put it
   in `check_link_resource.go` and the three `New*` constructors would sit
   naturally next to it.
3. **What about `internal/client/`?** That one is fine as-is. The HTTP client
   is genuinely a separate concern and several reference providers keep it
   separate (e.g., `vault`, `cloudflare`).
4. **What `aws`, `azurerm`, `google` do** is *not* what you should copy —
   those are huge providers organized by service into per-service packages.
   For a 7-resource provider, the scaffolding layout is right.

Migration is mostly mechanical (`git mv` + change `package resources` to
`package provider`, drop the cross-package imports, fix the test packages).
Budget: 30–60 minutes.

---

## 2. Resource ↔ data source parity

### Imports
All 7 resources implement `ImportState` ✓. But two are partial:

- **`serverguard24_check`**: import sets only `id`, then `Read` populates the
  rest. Per the comment in `internal/acceptance/check_test.go:44-49`,
  `ImportStateVerify` is intentionally disabled because (a) `service_show`
  doesn't return `host_id` and (b) Create stores plan values, but Read
  populates more fields → state mismatch right after Create. **This means
  users importing existing checks see a noisy plan-diff.** Fix: have `Create`
  call `refreshFromAPI` after creating, like the AWS provider does, so
  post-Create state matches post-Read state.
- **`serverguard24_downtime`**: import only sets `id` (no `downtime_show` API),
  users must add all attributes to config and re-apply. Documented, but it's a
  sharp edge.

### Parity gap (the real issue)
You have data sources only for things you *can't* manage as resources
(`groups`, `distributors`, `integrations`, `nodes`). You have **no data source
for the things you *do* manage**:

| Resource | Data source? |
|---|---|
| `serverguard24_host` | ❌ missing |
| `serverguard24_contact` | ❌ missing |
| `serverguard24_check` | ❌ missing |

This matters for OSS adoption because of the existing UX problem your README
already acknowledges (`README.md:101-105`): users have to enable "Show Object
IDs" in the dashboard and copy them by hand. With data sources by name they
could write:

```hcl
data "serverguard24_host" "existing" {
  name = "web.example.com"
}

resource "serverguard24_check" "http" {
  host_id = data.serverguard24_host.existing.id
  ...
}
```

The API has `get_hosts`, `get_contacts`, and (presumably) a way to enumerate
checks — so the implementation cost is roughly one file per data source,
mirroring `groups.go`.

Ship those three before going OSS. Without them, every user's first 10 minutes
will be hand-copying IDs.

---

## 3. Bugs (correctness)

These are real, not stylistic. In rough priority order.

### 3.1 Notification flags can't be turned off
File: `internal/resources/check.go:1186-1190`

```go
setOptionalBoolAsInt := func(key string, v types.Bool) {
    if !v.IsNull() && !v.IsUnknown() && v.ValueBool() {
        cmd[key] = 1
    }
}
```

The comment says "the API returns 500 when the field is sent as 0; omitting
equals false". That's fine for **create**. For **update** it's broken:

1. User sets `email_critical = true`, applies. API stores 1.
2. User changes to `email_critical = false`, applies. The Update call **omits
   the key**. The API still has 1.
3. Refresh reads 1 from `email_critical`. State goes back to `true`.
   **Perpetual diff.**

Same problem affects every notification flag (`email_*`, `sms_*`, `push_*`,
`http_*`, `phone_*`, 20 attributes total) and `ipv6`, `invert-regex`,
`redirect`, `smtp_tls`, `check_cert`, `ssl`, `performance`. ~30 fields total.

Fixes, in order of preference:

1. Find what the API actually wants for "false" (try `cmd[key] = false`,
   `"0"`, `"off"`, omitted in CREATE but explicit in EDIT). Worth a 30-minute
   API-side test.
2. If nothing works, document the limitation prominently and emit a plan-time
   warning when a flag is changed from true → false.
3. Make those attributes `RequiresReplace` if you can't make false stick.

### 3.2 String fields can't be cleared
File: `internal/resources/contact.go:110-112`, `internal/resources/host.go:97-98`

`setOptionalString` skips null/unknown:

```go
func setOptionalString(cmd client.Request, key string, v types.String) {
    if !v.IsNull() && !v.IsUnknown() {
        cmd[key] = v.ValueString()
    }
}
```

If a user removes `email = "..."` from a contact, the Update call doesn't send
`email`, so the API keeps the old value. Same for `mobile`, `network`, contact,
and most check string params. Fix: in **update** path, send `cmd[key] = ""`
when the value is null but state was non-null — or document that field-clearing
is unsupported.

### 3.3 Error classification by `strings.Contains(err.Error(), …)`
File: `internal/client/client.go:129-152`

```go
func IsNotFound(err error) bool {
    return strings.Contains(err.Error(), "API error 210")
}
```

If `Do()`'s error format ever changes (or if a wrapped error embeds the literal
"API error 210" in some unrelated way), all classification breaks silently.
Use a typed error:

```go
type APIError struct { Code, Message string }
func (e *APIError) Error() string { return fmt.Sprintf("API error %s: %s", e.Code, e.Message) }

func IsNotFound(err error) bool {
    var ae *APIError
    return errors.As(err, &ae) && ae.Code == "210"
}
```

Small refactor that eliminates a whole class of bug.

### 3.4 No context propagation in the HTTP client
File: `internal/client/client.go:65`

`Do(cmd Request)` doesn't take `context.Context`. That means `terraform apply`
cancellation (Ctrl+C) cannot interrupt an in-flight HTTP request — the caller
will block until the 30s timeout. Add `DoCtx(ctx, cmd)` and pass `ctx` from
every Create/Read/Update/Delete. Resource methods already receive a `ctx`.

### 3.5 No retries for transient failures
File: `internal/client/client.go`

Single attempt, no backoff. Network blips, 502s from a load balancer, brief
rate-limits — all fail terraform applies. `hashicorp/go-retryablehttp` is
already an indirect dep; wrap the client in it for idempotent reads at minimum.

### 3.6 Downtime adoption can pick the wrong record
File: `internal/resources/downtime.go:213-225, 405-423`

When create returns 310 ("already exists"), `findDowntimeByDescription` returns
the **first** match by description. If a user creates two downtimes with
`description = "Planned maintenance"` (one expired, one current), the wrong one
will be adopted into state. Match on `(description, hostname, start_time)` to
disambiguate.

### 3.7 `host.Read` is O(N) per host
File: `internal/resources/host.go:118-164`

`get_hosts` ignores the id filter and returns *every* host on the account. The
provider iterates to find the match. Per-host. So a `terraform refresh` on M
hosts = M API calls × N total hosts. Fine for 10. Painful for 1000. Either:

- Cache the full host list per provider invocation (`sync.Once` in the client),
  or
- Document that large fleets will be slow.

Same pattern in `internal/resources/contact.go:135-180`.

### 3.8 Acceptance tests don't verify destruction

No `CheckDestroy` in any of `host_test.go`, `check_test.go`, `contact_test.go`,
etc. A test passes if Create succeeds — but if Delete silently failed, you'd
never know. Concretely, this means failed test runs may leave orphan resources
in the test account. Add `CheckDestroy: testAccCheckHostDestroy` to each test
and implement the destroy verifier (call `get_hosts`, assert ID not present).
Plus `Sweepers` to clean up leftovers from previous broken runs.

### 3.9 Update calls `service_monitoring`/`service_notification` unconditionally
File: `internal/resources/check.go:1111-1118`

Every check Update issues two extra API calls even when `active`/`notification`
haven't changed. Cheap but wasteful, and triples the rate-limit pressure on
bulk updates. Skip when `plan.Active.Equal(state.Active)` and
`plan.Notification.Equal(state.Notification)`.

---

## 4. Open-source readiness — what's missing

Things a contributor visiting `github.com/dg-i/terraform-provider-serverguard24`
will look for first:

| Missing | Why it matters |
|---|---|
| `CONTRIBUTING.md` | How to dev/test/submit PRs. |
| `CODE_OF_CONDUCT.md` | GitHub flags repos that lack one. Use the Contributor Covenant. |
| `SECURITY.md` | Where to report vulns privately. Critical for a credential-handling provider. |
| `.github/dependabot.yml` | Auto-PRs for Go module + Action updates. |
| `.github/ISSUE_TEMPLATE/{bug_report,feature_request}.md` | Triage hygiene. |
| `.github/pull_request_template.md` | Same. |
| `CODEOWNERS` | Auto-assign reviewers. |
| Badges in `README.md` | Build status, Terraform Registry version, license, Go version. First thing visitors see. |
| `AGENTS.md` / `CLAUDE.md` symlink | Either rename to `CONTRIBUTING.md` or move to `docs/internal/` — Claude-specific guidance reads strangely in an OSS context. |

The CHANGELOG only has one entry (`0.1.2 (2026-04-27)`) but git log shows
`0.1.0` and `0.1.2` releases. Backfill or use
[Keep a Changelog](https://keepachangelog.com) format with an `Unreleased`
section.

---

## 5. Build / CI nits

1. **`go.mod` has `go 1.26.2`** but README says "Go >= 1.26.2 (for building
   from source)" and CLAUDE.md says "Go 1.21+". Pick one — current stable is
   1.23. Pinning to a future version like 1.26.2 means contributors can't even
   compile until they install a custom toolchain. I suspect this is a typo;
   double-check.
2. **`golangci-lint` versions diverge** between `.pre-commit-config.yaml`
   (`v2.11.4`) and `.github/workflows/test.yml` (`@latest`). Pin both to the
   same version, or you'll get "passes locally, fails in CI" surprises.
3. **`.golangci.yml` is bare** — only excludes errcheck/bodyclose for tests.
   Enable at least `gofmt`, `govet`, `staticcheck`, `gosec`, `misspell`,
   `unconvert`, `unused`. For a public provider, `gosec` is high-value.
4. **Add `govulncheck`** as a CI step — one-liner using
   `golang.org/x/vuln/cmd/govulncheck`.
5. **No race detector run.** Add `go test -race ./...` to CI.
6. **`testacc` job runs on every push to main and every same-repo PR.** Two
   parallel runs can race on resource names (the random suffix is 8 chars,
   collision probability is small but nonzero). Consider serializing with a
   `concurrency` group.

---

## 6. Code quality nits (non-blocking)

- **`check.go` is 1661 lines.** The encode (`buildCheckCommand`) and decode
  (`applyCheckData`) functions duplicate the field list across two ~250-line
  blocks. Each new param requires touching three places (schema, encode,
  decode) plus the `paramsSetFields` registry. A small table-driven approach
  (one `[]paramSpec{name, attr, encode, decode}` slice) would shrink it by
  50%+ and prevent encode/decode drift bugs.
- **`paramsSetFields` allocates a new slice on every call** (~110 entries).
  Called in `ValidateConfig` once per validation. Fine, but cache it via a
  package-level var.
- **`joinStrings(ss, sep)`** in `check.go:365-374` reimplements
  `strings.Join`. Use `strings.Join`.
- **`emptyList()` returns `types.ListValueMust`** with no diagnostic capture —
  `Must` panics on a malformed list, but the static value can't fail.
  Acceptable but consider.
- **The `useStateXxxModifier` types** are duplicated 5 times for String, Bool,
  Int64, Float64, Object. The plugin framework's `default` modifiers cover
  most of these; consider whether you need all five or whether you can get
  away with `*default.StaticString("")` etc.
- **Sensitive markings:** `password`, `pop_password`, `smtp_password`,
  `basicauth_password`, `token` are marked `Sensitive: true` ✓. But
  `auth_type`, `username`, `pop_user`, `login` aren't (probably right — those
  aren't secrets).

---

## Recommended ordering

If prioritizing for an OSS launch:

1. **Block release on:** §3.1 (notification flags), §3.2 (string field
   clearing), §3.3 (typed errors), §2 (host/contact/check data sources), §3.8
   (CheckDestroy + sweepers).
2. **Before launch announce:** §1 (restructure to `internal/provider/`), §4
   (CONTRIBUTING/SECURITY/etc.), badges in README, §5.1 (Go version), §5.2
   (linter version pinning).
3. **First post-launch milestone:** §3.4 (context), §3.5 (retries), §3.6
   (downtime adoption), §3.7 (O(N) reads), §3.9 (skip no-op updates).
4. **Second milestone:** §6 (refactor check.go encode/decode).

## Net assessment

Solid v0.1.x work — tests are real, GoReleaser/GPG is set up properly, the API
quirks are mostly handled, the link-resource factory is genuinely nice. The
above is what you'd polish before putting the company name on it.
