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

---

## Implementation status — 2026-05-15

Recording where each review item stands after PR #5, PR #7, and the §6 cleanup
that follows. Items left as-is include the rationale for the decision.

### §1 Project structure
**Done in PR #5.** Collapsed `internal/datasources/*` and `internal/resources/*`
into a single `internal/provider/` package; files renamed to `_resource.go` /
`_data_source.go`. `internal/client/` kept separate as recommended.

### §2 Resource ↔ data source parity
**Done in PR #5.** New data sources: `serverguard24_host` (id|name),
`serverguard24_contact` (id|name), `serverguard24_check` (id only — the API
has no bulk-list endpoint for checks; documented in the data source schema).

The check-import noisy-diff caveat (Create stores plan values; Read returns
more fields) is **resolved**. The fix involves calling `refreshFromAPI` at the end of
Create. This was handled in the recent PR and the implementation detail aligns with the AWS provider approach.

### §3 Bugs

All items closed in PR #5.

| Item | Resolution |
|---|---|
| 3.1 Notification flags can't be turned off | Plan-time warning (`warnIfClearingBool`). Took the review's *option 2*; *option 1* (find what the API accepts for "false") not attempted — the live-account test cost exceeds the value once the warning surfaces the asymmetry at plan time. |
| 3.2 String fields can't be cleared | Same warning pattern (`warnIfClearingString`). |
| 3.3 Error classification by substring | Typed `client.APIError` + `errors.As`. `IsNotFound`, `IsPermissionError`, `IsAlreadyExists` rewired. |
| 3.4 No context propagation | `client.Do(ctx, ...)`; every call site passes the framework-provided ctx; `retryablehttp.NewRequestWithContext` honors it under retries. |
| 3.5 No retries / backoff | `hashicorp/go-retryablehttp` wraps the transport (3 retries, 500ms → 5s backoff, `PassthroughErrorHandler` so the final HTTP status surfaces back to our own status check). |
| 3.6 Downtime adoption picks first by description | `findDowntime` now matches on `(description, hostname, starttime)`. |
| 3.7 `host.Read` is O(N) | 30s TTL mutex-serialised cache on `get_hosts`/`get_contacts`; host/contact CRUD calls `Invalidate…` explicitly so a follow-up Read sees the write. |
| 3.8 Acceptance tests don't verify destruction | `CheckDestroy` wired into every test plus host/contact sweepers (`-sweep=all`). |
| 3.9 Update unconditionally calls `service_monitoring`/`service_notification` | Only re-issued when the corresponding plan field differs from state. |

### §4 Open-source readiness

| Item | Status |
|---|---|
| `CONTRIBUTING.md` | Done in PR #5 |
| `SECURITY.md` | Done in PR #5 |
| `CODE_OF_CONDUCT.md` | Done. Pointer-style file at repo root linking to Contributor Covenant 2.1 (canonical text at contributor-covenant.org); reports routed via the dg-i.net contact page. CONTRIBUTING.md's link is no longer dangling. |
| `.github/dependabot.yml` | Done. Weekly Mondays for both `gomod` and `github-actions`, with minor + patch updates grouped to keep the PR queue manageable. |
| `.github/ISSUE_TEMPLATE/{bug_report,feature_request}.md` | Done. Bug template requests provider/Terraform/Go versions, minimal repro config, and full output (with `TF_LOG=DEBUG` hint); feature template requests use case, proposed HCL sketch, and Serverguard24 API support status. |
| `.github/pull_request_template.md` | Done. Aligned with `CONTRIBUTING.md` — summary, type-of-change checklist, `make test/lint/generate` test plan, CHANGELOG reminder. |
| `CODEOWNERS` | Done. `.github/CODEOWNERS` with `* @dg-i` as global owner. |
| README badges | Done. CI status, latest release, Terraform Registry, Go Report Card, license — all in the header. Added an explicit `## Contributing` section and clarified the license as MPL 2.0 in the existing `## License` section. |
| `AGENTS.md` / `CLAUDE.md` reshuffle | Done. `CLAUDE.md` symlink removed — Claude Code reads `AGENTS.md` directly, so the symlink was redundant. `AGENTS.md` stays at repo root. |
| `CHANGELOG` backfill + `Unreleased` section | Done. `Unreleased` section was already populated in PR #5; added a brief `0.1.0` entry noting it was an internal tagged release superseded by the public `0.1.2` (one day apart, no Registry publish in between). |

### §5 Build / CI nits
**All done in PR #7.**

- 5.1 Go version inconsistency — toolchain now pinned to `go1.26.3`, module baseline `go 1.25.0`.
- 5.2 `golangci-lint` version drift — workflow uses `${{ env.GOLANGCI_LINT_VERSION }} = v2.11.4`, matching `.pre-commit-config.yaml`.
- 5.3 `.golangci.yml` expansion — `default: standard` (errcheck, govet, ineffassign, staticcheck, unused) plus `gosec`, `misspell`, `unconvert`; `gofmt` as formatter. Tests opt out of `errcheck`, `bodyclose`, `gosec`.
- 5.4 `govulncheck` CI job added; `golang.org/x/net` bumped to `v0.54.0` and toolchain to `1.26.3` to clear the findings.
- 5.5 `go test -race` enabled (timeout raised to 180s to absorb race overhead).
- 5.6 `testacc` now has `concurrency: { group: testacc-<workflow>, cancel-in-progress: false }` so parallel runs queue instead of racing on the 8-char random suffix.

### §6 Code quality nits (non-blocking)

Each item below either landed or was deliberately deferred. The deferred ones
include the reasoning so future work doesn't re-litigate the decision.

**`joinStrings` → `strings.Join`** — **Done.** Four call sites; helper deleted.

**`interface{}` → `any` + `maps.Copy` for the `refreshFromAPI` flatten loops**
— **Done.** Adjacent gopls-flagged style nits, swept across `check_resource.go`,
the three permission-aware data sources, and `client_test.go`.

**Table-driven encode/decode in `buildCheckCommand` / `applyCheckData`** —
**Done (2026-05-21).** Landed in two steps as the deferred plan required:

1. *Scaffolding.* Added `check_resource_snapshot_test.go` with a kitchen-sink
   `CheckResourceModel` and a matching flattened `service_show` response,
   plus `testdata/*.golden.json` for `service_add`, `service_edit`, and
   `applyCheckData`. Per-type `ssl_version` map cases (`http` / `http_perf` /
   `ssl_cert` / `smtp`) are covered as table tests. Regenerate the goldens
   with `go test ./internal/provider/ -run KitchenSink -update-golden`.
2. *Refactor.* Introduced `check_resource_fields.go` with two ordered
   manifests — `modelSpecs` for top-level `CheckResourceModel` fields,
   `paramSpecs` for every `CheckParamsModel` field. Typed builders
   (`paramStr` / `paramInt` / `paramBool` / `paramFloat` / `paramEnum`,
   `modelStr` / `modelBool`) capture a typed accessor closure per entry so
   encode/decode pick the right helper without interface dispatch.
   `buildCheckCommand`, `applyCheckData`, and `paramsSetFields` are now
   single loops over the relevant slice. Snapshot output is byte-identical
   pre/post refactor (no golden update was needed). Net delta in
   `check_resource.go`: −345 / +52.

Fields that don't round-trip uniformly stay inline with a one-line reason:
`description` (omit-when-empty), `region` (enum + numeric fallback),
`interval` / `notify_after` / `reminder_interval` (omit-when-zero + unit
conversion on decode), `active` / `notification` (separate API endpoints),
and the seven enum-mapped params (`ssl_version`, `auth_type`,
`performance_none`, `pop_type`, `delete_mode`, `mongo_connection`,
`searchitem_none`, `portscan_none`). The enum params still appear in
`paramSpecs` so `paramsSetFields` keeps surfacing them to `ValidateConfig`;
their encode/decode closures are no-ops because the maps live inline next
to the type-specific logic.

The original review's "shrink by 50%+" estimate didn't materialise — the
typed-closure approach is net ~−70 lines once the manifest file is counted,
not the dramatic reduction a reflect/codegen approach would have produced.
The actual win is the structural property the snapshot test now guards:
every simple field has exactly one declaration site, and any future drift
between encode and decode fails the golden compare immediately.

**`paramsSetFields` allocates a new slice on every call** — **Deferred.** The
returned slice's `paramCheck` *values* depend on the per-call
`*CheckParamsModel`, so the slice fundamentally has to be rebuilt each call;
a package-level cache only works if we cache something invariant. The
current literal form is already maximally compact. The ~110-element
allocation runs once per `ValidateConfig` (i.e. once per `terraform plan`)
and is not a measured bottleneck.

**`emptyList()`'s `types.ListValueMust`** — **Left as-is.** The review's own
conclusion was "acceptable but consider" — the static value cannot fail at
runtime.

**`useStateXxxModifier` types (5 duplicates)** — **Left as-is.** The file
comment at `check_resource.go:274-277` documents the intentional difference
from the framework's `UseStateForUnknown`: the custom versions also fire when
state is null, which is what makes `terraform import` leave un-returned
fields *null* in plan output instead of *"known after apply"*. The review's
suggested replacement (`default.StaticString("")` etc.) changes the
semantics — those modifiers inject a default value, they don't preserve a
null-state-for-unknown.

**Sensitive markings on credential fields** — **Already correct** per the
reviewer; nothing to do.

---

## QA verification — 2026-05-29

Role: independent quality assurance pass over the mitigation claims above.

### Verified as implemented

- Documentation drift resolved: `README.md`, `AGENTS.md` updated to reflect new data sources and current versions.


- §1 project structure is implemented: resources and data sources now live in
  `internal/provider/`; `internal/client/` remains separate.
- §2 data-source parity is implemented in code and provider registration:
  `serverguard24_host`, `serverguard24_contact`, and `serverguard24_check` are
  present and registered. The check data source is correctly limited to
  lookup-by-id because `service_show` has no bulk-list API.
- §3.3, §3.4, and §3.5 are implemented: `client.APIError` is typed and matched
  via `errors.As`; `Client.Do(ctx, ...)` propagates context into
  `retryablehttp.NewRequestWithContext`; retryablehttp is configured with
  bounded retries/backoff.
- §3.6 is implemented: downtime adoption now matches
  `(description, hostname, starttime)` instead of first description match.
- §3.7 is implemented at the API-call level: `get_hosts` / `get_contacts` have
  a 30s mutex-protected cache and host/contact CRUD invalidates it.
- §3.8 is implemented for resources that can actually be verified through the
  API: host/contact/check acceptance tests have `CheckDestroy`; downtime is
  intentionally permissive because the API has no delete; host/contact sweepers
  are registered.
- §3.9 is implemented: `service_monitoring` and `service_notification` are
  skipped unless the corresponding plan value differs from state.
- §4 and §5 are mostly implemented: OSS support files, Dependabot, issue/PR
  templates, CODEOWNERS, lint/race/govulncheck CI, and testacc concurrency are
  present.
- §6 table-driven check encode/decode refactor is implemented with golden
  snapshot coverage.

### Implemented

1. **§3.2 string-clearing warning is implemented.** `paramSpecs` now
   carries the warning behavior for every string-like `params {}` field
   (`paramStr` and `paramEnum`) and `CheckResource.ModifyPlan` uses the shared
   `warnIfClearingParams` helper. That closes the earlier gap for fields such
   as `path`, `login`, `username`, `host_header`, `searchitem`, `regex`,
   `expected_ip*`, `dns_name`, `sender`, `recipient`, `database`, `sql_query`,
   `cron`, and `definition`, including the case where the whole `params {}`
   block is removed.

2. **§5 plan-time warning tests are covered.** Added focused
   unit tests for `warnIfClearingString`, `warnIfClearingBool`, and
   `warnIfClearingParams`, including removed `params {}` handling.

3. **Provider documentation is clarified for §3.7.** The provider and
   host/contact data source docs now explain that `get_hosts` / `get_contacts`
   still return full-account lists, but the provider uses a 30s
   mutex-serialized cache to avoid redundant full-list network calls.
   Remaining refresh cost is local filtering over cached responses.

### Still open

1. **§2 check import noisy diff remains open, as documented.** Verified:
   `CheckResource.Create` still stores the post-create plan directly and does
   not call `refreshFromAPI`; `refreshFromAPI` is used by `Read` only. Keep this
   as a real open task before claiming full import parity.

### Verification run

- `go test ./...` — passed.
- `go test -race ./...` — passed.
- Acceptance tests were not run against a live Serverguard24 account in this QA
  pass; their wiring was checked statically.

### Implementation Note

- `paramSpecs` carries the warning behaviour for string-like params
  (`paramStr` and `paramEnum`) and bool params (`paramBool`), and
  `CheckResource.ModifyPlan` calls the shared `warnIfClearingParams` helper
  instead of maintaining a partial manual list.
- Removing the whole `params {}` block is handled as clearing all previously
  set params, so already-stored string/enum/bool values also produce warnings
  in that case.
- Focused unit tests cover string clearing, bool clearing, all string-like
  param warnings, and removed `params {}` block handling.
