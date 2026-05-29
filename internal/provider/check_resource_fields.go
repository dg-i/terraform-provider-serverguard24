package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
)

// Field manifest for the check resource. Every "simple" round-trippable
// attribute (model field ↔ API key with no enum or per-type logic) appears
// exactly once in one of the two ordered slices below — modelSpecs for
// top-level CheckResourceModel fields, paramSpecs for CheckParamsModel
// fields. buildCheckCommand, applyCheckData, and paramsSetFields each
// iterate the relevant slice with a single loop.
//
// Special cases — description (omit-when-empty), region (enum + numeric
// fallback), interval/notify_after/reminder (omit-when-zero, unit
// conversion on decode), Active/Notification (separate API endpoint), and
// the eight enum-mapped params (ssl_version etc.) — remain inline in
// buildCheckCommand / applyCheckData. The enum params still appear in
// paramSpecs so paramsSetFields surfaces them to ValidateConfig; their
// encode/decode closures are no-ops because the maps live inline next to
// the type-specific logic.

// fieldSpec ties a schema attribute to its three round-trip operations.
// The builders below construct one of these per attribute; each is captured
// via a closure over the typed model accessor so encode/decode can use the
// correct setOptional*/decode* helper without an interface dispatch.
type fieldSpec[S any] struct {
	name      string
	encode    func(cmd client.Request, s *S)
	decode    func(s *S, raw map[string]any)
	isSet     func(s *S) (set, unknown bool)
	warnClear func(diags *diag.Diagnostics, p path.Path, state, plan *S)
}

// ── Builders ──────────────────────────────────────────────────────────────
//
// One builder per (kind, source). The optional `api` argument is variadic so
// the common case (api == name) needs no extra ceremony.

func paramStr(n string, get func(*CheckParamsModel) *types.String, api ...string) fieldSpec[CheckParamsModel] {
	key := apiOr(n, api)
	return fieldSpec[CheckParamsModel]{
		name:   n,
		encode: func(cmd client.Request, p *CheckParamsModel) { setOptionalString(cmd, key, *get(p)) },
		decode: func(p *CheckParamsModel, raw map[string]any) { *get(p) = decodeString(raw, key, *get(p)) },
		isSet:  isSetGeneric(get),
		warnClear: func(diags *diag.Diagnostics, p path.Path, state, plan *CheckParamsModel) {
			warnIfClearingString(diags, p, *get(state), *get(plan))
		},
	}
}

func paramInt(n string, get func(*CheckParamsModel) *types.Int64, api ...string) fieldSpec[CheckParamsModel] {
	key := apiOr(n, api)
	return fieldSpec[CheckParamsModel]{
		name:   n,
		encode: func(cmd client.Request, p *CheckParamsModel) { setOptionalInt(cmd, key, *get(p)) },
		decode: func(p *CheckParamsModel, raw map[string]any) { *get(p) = decodeInt(raw, key, *get(p)) },
		isSet:  isSetGeneric(get),
	}
}

func paramFloat(n string, get func(*CheckParamsModel) *types.Float64, api ...string) fieldSpec[CheckParamsModel] {
	key := apiOr(n, api)
	return fieldSpec[CheckParamsModel]{
		name:   n,
		encode: func(cmd client.Request, p *CheckParamsModel) { setOptionalFloat(cmd, key, *get(p)) },
		decode: func(p *CheckParamsModel, raw map[string]any) { *get(p) = decodeFloat(raw, key, *get(p)) },
		isSet:  isSetGeneric(get),
	}
}

func paramBool(n string, get func(*CheckParamsModel) *types.Bool, api ...string) fieldSpec[CheckParamsModel] {
	key := apiOr(n, api)
	return fieldSpec[CheckParamsModel]{
		name:   n,
		encode: func(cmd client.Request, p *CheckParamsModel) { setOptionalBoolAsInt(cmd, key, *get(p)) },
		decode: func(p *CheckParamsModel, raw map[string]any) { *get(p) = decodeBool(raw, key, *get(p)) },
		isSet:  isSetGeneric(get),
		warnClear: func(diags *diag.Diagnostics, p path.Path, state, plan *CheckParamsModel) {
			warnIfClearingBool(diags, p, *get(state), *get(plan))
		},
	}
}

// paramEnum is for params whose round-trip needs an inline enum map. The
// encode/decode closures are no-ops; only isSet matters (for paramsSetFields).
func paramEnum(n string, get func(*CheckParamsModel) *types.String) fieldSpec[CheckParamsModel] {
	return fieldSpec[CheckParamsModel]{
		name:   n,
		encode: func(client.Request, *CheckParamsModel) {},
		decode: func(*CheckParamsModel, map[string]any) {},
		isSet:  isSetGeneric(get),
		warnClear: func(diags *diag.Diagnostics, p path.Path, state, plan *CheckParamsModel) {
			warnIfClearingString(diags, p, *get(state), *get(plan))
		},
	}
}

func warnIfClearingParams(diags *diag.Diagnostics, state, plan *CheckParamsModel) {
	if state == nil {
		return
	}
	if plan == nil {
		plan = &CheckParamsModel{}
	}
	for _, s := range paramSpecs {
		if s.warnClear == nil {
			continue
		}
		s.warnClear(diags, path.Root("params").AtName(s.name), state, plan)
	}
}

func modelStr(n string, get func(*CheckResourceModel) *types.String) fieldSpec[CheckResourceModel] {
	return fieldSpec[CheckResourceModel]{
		name:   n,
		encode: func(cmd client.Request, m *CheckResourceModel) { setOptionalString(cmd, n, *get(m)) },
		decode: func(m *CheckResourceModel, raw map[string]any) { *get(m) = decodeString(raw, n, *get(m)) },
		isSet:  isSetGeneric(get),
	}
}

func modelBool(n string, get func(*CheckResourceModel) *types.Bool) fieldSpec[CheckResourceModel] {
	return fieldSpec[CheckResourceModel]{
		name:   n,
		encode: func(cmd client.Request, m *CheckResourceModel) { setOptionalBoolAsInt(cmd, n, *get(m)) },
		decode: func(m *CheckResourceModel, raw map[string]any) { *get(m) = decodeBool(raw, n, *get(m)) },
		isSet:  isSetGeneric(get),
	}
}

func apiOr(n string, api []string) string {
	if len(api) > 0 {
		return api[0]
	}
	return n
}

// isSetGeneric returns the canonical isSet closure for any `types.X` with
// IsNull / IsUnknown methods. (Generic helper so the per-kind builders share
// a single line for this concern.)
func isSetGeneric[S any, T interface {
	IsNull() bool
	IsUnknown() bool
}](get func(*S) *T) func(*S) (bool, bool) {
	return func(s *S) (bool, bool) {
		v := *get(s)
		return !v.IsNull() && !v.IsUnknown(), v.IsUnknown()
	}
}

// ── Manifests ─────────────────────────────────────────────────────────────

// modelSpecs covers the top-level CheckResourceModel fields whose round-trip
// is a single setOptional*/decode* call. Order matches declaration order in
// the CheckResourceModel struct.
var modelSpecs = []fieldSpec[CheckResourceModel]{
	modelStr("shortname", func(m *CheckResourceModel) *types.String { return &m.Shortname }),
	modelStr("address", func(m *CheckResourceModel) *types.String { return &m.Address }),
	modelBool("ipv6", func(m *CheckResourceModel) *types.Bool { return &m.IPv6 }),
	modelStr("message", func(m *CheckResourceModel) *types.String { return &m.Message }),

	modelBool("email_critical", func(m *CheckResourceModel) *types.Bool { return &m.EmailCritical }),
	modelBool("email_warning", func(m *CheckResourceModel) *types.Bool { return &m.EmailWarning }),
	modelBool("email_ok", func(m *CheckResourceModel) *types.Bool { return &m.EmailOk }),
	modelBool("email_remind", func(m *CheckResourceModel) *types.Bool { return &m.EmailRemind }),
	modelBool("sms_critical", func(m *CheckResourceModel) *types.Bool { return &m.SmsCritical }),
	modelBool("sms_warning", func(m *CheckResourceModel) *types.Bool { return &m.SmsWarning }),
	modelBool("sms_ok", func(m *CheckResourceModel) *types.Bool { return &m.SmsOk }),
	modelBool("sms_remind", func(m *CheckResourceModel) *types.Bool { return &m.SmsRemind }),
	modelBool("push_critical", func(m *CheckResourceModel) *types.Bool { return &m.PushCritical }),
	modelBool("push_warning", func(m *CheckResourceModel) *types.Bool { return &m.PushWarning }),
	modelBool("push_ok", func(m *CheckResourceModel) *types.Bool { return &m.PushOk }),
	modelBool("push_remind", func(m *CheckResourceModel) *types.Bool { return &m.PushRemind }),
	modelBool("http_critical", func(m *CheckResourceModel) *types.Bool { return &m.HttpCritical }),
	modelBool("http_warning", func(m *CheckResourceModel) *types.Bool { return &m.HttpWarning }),
	modelBool("http_ok", func(m *CheckResourceModel) *types.Bool { return &m.HttpOk }),
	modelBool("http_remind", func(m *CheckResourceModel) *types.Bool { return &m.HttpRemind }),
	modelBool("phone_critical", func(m *CheckResourceModel) *types.Bool { return &m.PhoneCritical }),
	modelBool("phone_warning", func(m *CheckResourceModel) *types.Bool { return &m.PhoneWarning }),
	modelBool("phone_ok", func(m *CheckResourceModel) *types.Bool { return &m.PhoneOk }),
	modelBool("phone_remind", func(m *CheckResourceModel) *types.Bool { return &m.PhoneRemind }),
}

// paramSpecs covers every CheckParamsModel field. Order matches the struct
// declaration. Enum-mapped fields use paramEnum — their encode/decode are
// no-ops and the mapping logic lives inline in buildCheckCommand /
// applyCheckData. The third argument to paramStr/paramInt/paramBool is an
// API-key override for the few fields whose schema name differs from the
// wire name.
var paramSpecs = []fieldSpec[CheckParamsModel]{
	paramStr("path", func(p *CheckParamsModel) *types.String { return &p.Path }),
	paramInt("timeout", func(p *CheckParamsModel) *types.Int64 { return &p.Timeout }),
	paramInt("port", func(p *CheckParamsModel) *types.Int64 { return &p.Port }),
	paramBool("ssl", func(p *CheckParamsModel) *types.Bool { return &p.Ssl }),
	paramEnum("ssl_version", func(p *CheckParamsModel) *types.String { return &p.SslVersion }),
	paramBool("check_cert", func(p *CheckParamsModel) *types.Bool { return &p.CheckCert }),
	paramEnum("auth_type", func(p *CheckParamsModel) *types.String { return &p.AuthType }),
	paramStr("login", func(p *CheckParamsModel) *types.String { return &p.Login }),
	paramStr("password", func(p *CheckParamsModel) *types.String { return &p.Password }),
	paramStr("username", func(p *CheckParamsModel) *types.String { return &p.Username }),
	paramStr("useragent", func(p *CheckParamsModel) *types.String { return &p.Useragent }),
	paramStr("host_header", func(p *CheckParamsModel) *types.String { return &p.HostHeader }),
	paramStr("searchitem", func(p *CheckParamsModel) *types.String { return &p.Searchitem }),
	paramStr("method", func(p *CheckParamsModel) *types.String { return &p.Method }),
	paramStr("postvars", func(p *CheckParamsModel) *types.String { return &p.Postvars }),
	paramStr("regex", func(p *CheckParamsModel) *types.String { return &p.Regex }),
	paramBool("invert_regex", func(p *CheckParamsModel) *types.Bool { return &p.InvertRegex }, "invert-regex"),
	paramStr("header_regex", func(p *CheckParamsModel) *types.String { return &p.HeaderRegex }, "header-regex"),
	paramInt("min_size", func(p *CheckParamsModel) *types.Int64 { return &p.MinSize }),
	paramInt("max_size", func(p *CheckParamsModel) *types.Int64 { return &p.MaxSize }),
	paramStr("addheader1", func(p *CheckParamsModel) *types.String { return &p.Addheader1 }),
	paramStr("addheader2", func(p *CheckParamsModel) *types.String { return &p.Addheader2 }),
	paramStr("addheader3", func(p *CheckParamsModel) *types.String { return &p.Addheader3 }),
	paramStr("content_type", func(p *CheckParamsModel) *types.String { return &p.ContentType }),
	paramInt("status_code", func(p *CheckParamsModel) *types.Int64 { return &p.StatusCode }),
	paramBool("redirect", func(p *CheckParamsModel) *types.Bool { return &p.Redirect }),
	paramFloat("warning_throughput", func(p *CheckParamsModel) *types.Float64 { return &p.WarningThroughput }),
	paramFloat("warning_connect", func(p *CheckParamsModel) *types.Float64 { return &p.WarningConnect }),
	paramFloat("warning_roundtrip", func(p *CheckParamsModel) *types.Float64 { return &p.WarningRoundtrip }),
	paramFloat("critical_throughput", func(p *CheckParamsModel) *types.Float64 { return &p.CriticalThroughput }),
	paramFloat("critical_connect", func(p *CheckParamsModel) *types.Float64 { return &p.CriticalConnect }),
	paramFloat("critical_roundtrip", func(p *CheckParamsModel) *types.Float64 { return &p.CriticalRoundtrip }),
	paramEnum("performance_none", func(p *CheckParamsModel) *types.String { return &p.PerformanceNone }),
	paramStr("protocol", func(p *CheckParamsModel) *types.String { return &p.Protocol }),
	paramInt("validity_warning", func(p *CheckParamsModel) *types.Int64 { return &p.ValidityWarning }),
	paramInt("validity_critical", func(p *CheckParamsModel) *types.Int64 { return &p.ValidityCritical }),
	paramStr("different_ip", func(p *CheckParamsModel) *types.String { return &p.DifferentIP }),
	paramStr("expected_ip1", func(p *CheckParamsModel) *types.String { return &p.ExpectedIP1 }),
	paramStr("expected_ip2", func(p *CheckParamsModel) *types.String { return &p.ExpectedIP2 }),
	paramStr("expected_ip3", func(p *CheckParamsModel) *types.String { return &p.ExpectedIP3 }),
	paramStr("expected_ip4", func(p *CheckParamsModel) *types.String { return &p.ExpectedIP4 }),
	paramStr("expected_iplink", func(p *CheckParamsModel) *types.String { return &p.ExpectedIPLink }),
	paramStr("dns_server", func(p *CheckParamsModel) *types.String { return &p.DnsServer }),
	paramStr("dns_record", func(p *CheckParamsModel) *types.String { return &p.DnsRecord }),
	paramStr("dns_auth", func(p *CheckParamsModel) *types.String { return &p.DnsAuth }),
	paramInt("response_warning", func(p *CheckParamsModel) *types.Int64 { return &p.ResponseWarning }),
	paramInt("response_critical", func(p *CheckParamsModel) *types.Int64 { return &p.ResponseCritical }, "response_critial"),
	paramStr("dns_name", func(p *CheckParamsModel) *types.String { return &p.DNSName }),
	paramStr("from", func(p *CheckParamsModel) *types.String { return &p.From }),
	paramInt("smtp_port", func(p *CheckParamsModel) *types.Int64 { return &p.SmtpPort }),
	paramStr("sender", func(p *CheckParamsModel) *types.String { return &p.Sender }),
	paramStr("recipient", func(p *CheckParamsModel) *types.String { return &p.Recipient }),
	paramEnum("pop_type", func(p *CheckParamsModel) *types.String { return &p.PopType }),
	paramStr("pop_host", func(p *CheckParamsModel) *types.String { return &p.PopHost }),
	paramStr("pop_user", func(p *CheckParamsModel) *types.String { return &p.PopUser }),
	paramStr("pop_password", func(p *CheckParamsModel) *types.String { return &p.PopPassword }),
	paramEnum("delete_mode", func(p *CheckParamsModel) *types.String { return &p.DeleteMode }),
	paramInt("time_warning", func(p *CheckParamsModel) *types.Int64 { return &p.TimeWarning }),
	paramInt("time_critical", func(p *CheckParamsModel) *types.Int64 { return &p.TimeCritical }),
	paramStr("smtp_user", func(p *CheckParamsModel) *types.String { return &p.SmtpUser }),
	paramStr("smtp_password", func(p *CheckParamsModel) *types.String { return &p.SmtpPassword }),
	paramBool("smtp_tls", func(p *CheckParamsModel) *types.Bool { return &p.SmtpTls }),
	paramStr("smtp_header1", func(p *CheckParamsModel) *types.String { return &p.SmtpHeader1 }),
	paramStr("database", func(p *CheckParamsModel) *types.String { return &p.Database }),
	paramStr("sql_query", func(p *CheckParamsModel) *types.String { return &p.SQLQuery }),
	paramEnum("mongo_connection", func(p *CheckParamsModel) *types.String { return &p.MongoConnection }),
	paramStr("searchitem_ok", func(p *CheckParamsModel) *types.String { return &p.SearchItemOk }),
	paramStr("searchitem_warning", func(p *CheckParamsModel) *types.String { return &p.SearchItemWarning }),
	paramStr("searchitem_critical", func(p *CheckParamsModel) *types.String { return &p.SearchItemCritical }),
	paramEnum("searchitem_none", func(p *CheckParamsModel) *types.String { return &p.SearchItemNone }),
	paramStr("basicauth_login", func(p *CheckParamsModel) *types.String { return &p.BasicauthLogin }),
	paramStr("basicauth_password", func(p *CheckParamsModel) *types.String { return &p.BasicauthPassword }),
	paramStr("debug_addon", func(p *CheckParamsModel) *types.String { return &p.DebugAddon }),
	paramStr("command", func(p *CheckParamsModel) *types.String { return &p.Command }, "cmd"),
	paramStr("token", func(p *CheckParamsModel) *types.String { return &p.Token }),
	paramInt("silent", func(p *CheckParamsModel) *types.Int64 { return &p.Silent }),
	paramStr("cron", func(p *CheckParamsModel) *types.String { return &p.Cron }),
	paramStr("timezone", func(p *CheckParamsModel) *types.String { return &p.Timezone }),
	paramStr("def", func(p *CheckParamsModel) *types.String { return &p.Def }),
	paramBool("performance", func(p *CheckParamsModel) *types.Bool { return &p.Performance }),
	paramStr("performance_name", func(p *CheckParamsModel) *types.String { return &p.PerformanceName }),
	paramStr("performance_regex", func(p *CheckParamsModel) *types.String { return &p.PerformanceRegex }),
	paramStr("performance_unit", func(p *CheckParamsModel) *types.String { return &p.PerformanceUnit }),
	paramInt("performance_digits", func(p *CheckParamsModel) *types.Int64 { return &p.PerformanceDigits }),
	paramStr("transaction_id", func(p *CheckParamsModel) *types.String { return &p.TransactionID }),
	paramStr("definition", func(p *CheckParamsModel) *types.String { return &p.Definition }),
	paramStr("language", func(p *CheckParamsModel) *types.String { return &p.Language }),
	paramStr("suite_filter", func(p *CheckParamsModel) *types.String { return &p.SuiteFilter }),
	paramStr("port_range", func(p *CheckParamsModel) *types.String { return &p.PortRange }),
	paramEnum("portscan_none", func(p *CheckParamsModel) *types.String { return &p.PortscanNone }),
	paramStr("open_ports", func(p *CheckParamsModel) *types.String { return &p.OpenPorts }),
}

// ── Encode helpers ──
//
// setOptionalString lives in contact_resource.go (shared with the contact /
// host encoders); the others are check-specific because the API's
// reject-on-zero rules don't apply elsewhere.

// setOptionalBoolAsInt encodes types.Bool as `1` when true and omits the key
// otherwise. The Serverguard24 API returns HTTP 500 if a notification or
// other bool field is sent as `0`, so we never emit the false case.
func setOptionalBoolAsInt(cmd client.Request, key string, v types.Bool) {
	if !v.IsNull() && !v.IsUnknown() && v.ValueBool() {
		cmd[key] = 1
	}
}

func setOptionalFloat(cmd client.Request, key string, v types.Float64) {
	if !v.IsNull() && !v.IsUnknown() {
		cmd[key] = v.ValueFloat64()
	}
}

func setOptionalInt(cmd client.Request, key string, v types.Int64) {
	if !v.IsNull() && !v.IsUnknown() {
		cmd[key] = v.ValueInt64()
	}
}

// ── Decode helpers ──
//
// These mirror the per-call closures applyCheckData used to hold; living at
// package scope lets the field-spec closures call them.

func decodeString(raw map[string]any, key string, cur types.String) types.String {
	switch v := raw[key].(type) {
	case string:
		return types.StringValue(v)
	case float64:
		if v == float64(int64(v)) {
			return types.StringValue(fmt.Sprintf("%d", int64(v)))
		}
		return types.StringValue(fmt.Sprintf("%g", v))
	}
	return cur
}

func decodeBool(raw map[string]any, key string, cur types.Bool) types.Bool {
	switch v := raw[key].(type) {
	case float64:
		return types.BoolValue(v == 1)
	case bool:
		return types.BoolValue(v)
	}
	return cur
}

func decodeFloat(raw map[string]any, key string, cur types.Float64) types.Float64 {
	if v, ok := raw[key].(float64); ok {
		return types.Float64Value(v)
	}
	return cur
}

func decodeInt(raw map[string]any, key string, cur types.Int64) types.Int64 {
	if v, ok := raw[key].(float64); ok {
		return types.Int64Value(int64(v))
	}
	return cur
}
