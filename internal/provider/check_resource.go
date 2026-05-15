package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
)

// checkTypeByName maps human-readable type names to the API integer IDs.
var checkTypeByName = map[string]int64{
	"http":             11,
	"http_perf":        13,
	"site_perf":        15,
	"ssl_cert":         14,
	"domain":           91,
	"ping":             40,
	"ssh":              60,
	"ftp":              20,
	"tcp":              80,
	"dns":              90,
	"smtp":             51,
	"pop3":             50,
	"imap":             52,
	"email":            53,
	"blacklist":        54,
	"mysql":            70,
	"postgresql":       72,
	"mssql":            73,
	"mongodb":          74,
	"local_web":        110,
	"local":            100,
	"intranet":         101,
	"cron":             102,
	"system":           103,
	"transact_server":  120,
	"transact_browser": 121,
	"selenium":         122,
	"postman":          123,
	"port_scan":        130,
}

// checkTypeByID is the reverse of checkTypeByName, built at init time.
var checkTypeByID = func() map[int64]string {
	m := make(map[int64]string, len(checkTypeByName))
	for name, id := range checkTypeByName {
		m[id] = name
	}
	return m
}()

// regionByName maps human-readable region names (as used in Terraform config) to the
// API integer IDs. Both named strings and raw numeric strings are accepted on input;
// the canonical name form is always written back to state on read.
var regionByName = map[string]int64{
	"individual":      0,
	"europe":          1,
	"americas":        2,
	"europe_americas": 3,
	"asia_pacific":    4,
	"world":           5,
}

// regionByID is the reverse of regionByName, built at init time.
var regionByID = func() map[int64]string {
	m := make(map[int64]string, len(regionByName))
	for name, id := range regionByName {
		m[id] = name
	}
	return m
}()

// validRegions returns the sorted list of valid region name strings.
func validRegions() []string {
	names := make([]string, 0, len(regionByName))
	for name := range regionByName {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// sslVersion maps — the integer encoding of ssl_version differs per check type.
//
//   http / http_perf : 1=auto  2=tlsv1  3=sslv3
//   ssl_cert         : 0=auto  1=tlsv1  2=sslv3
//   smtp             : 0=none  1=tls    2=ssl
var (
	sslVersionHTTPByName    = map[string]int64{"auto": 1, "tlsv1": 2, "sslv3": 3}
	sslVersionHTTPByID      = map[int64]string{1: "auto", 2: "tlsv1", 3: "sslv3"}
	sslVersionSSLCertByName = map[string]int64{"auto": 0, "tlsv1": 1, "sslv3": 2}
	sslVersionSSLCertByID   = map[int64]string{0: "auto", 1: "tlsv1", 2: "sslv3"}
	sslVersionSMTPByName    = map[string]int64{"none": 0, "tls": 1, "ssl": 2}
	sslVersionSMTPByID      = map[int64]string{0: "none", 1: "tls", 2: "ssl"}
)

// sslVersionMaps returns the name→ID and ID→name conversion maps for a check type.
func sslVersionMaps(checkType string) (map[string]int64, map[int64]string) {
	switch checkType {
	case "ssl_cert":
		return sslVersionSSLCertByName, sslVersionSSLCertByID
	case "smtp":
		return sslVersionSMTPByName, sslVersionSMTPByID
	default: // http, http_perf
		return sslVersionHTTPByName, sslVersionHTTPByID
	}
}

// validSSLVersions returns the sorted valid ssl_version string values for a check type.
func validSSLVersions(checkType string) []string {
	nameToID, _ := sslVersionMaps(checkType)
	names := make([]string, 0, len(nameToID))
	for name := range nameToID {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// auth_type maps — http, http_perf: "none"/"basic"/"ntlm" ↔ API codes "0"/"b"/"n"
var authTypeByName = map[string]string{"none": "0", "basic": "b", "ntlm": "n"}
var authTypeByCode = map[string]string{"0": "none", "b": "basic", "n": "ntlm"}

// performance_none maps — http_perf, site_perf, local_web, cron: "unknown"/"critical" ↔ 0/1
var performanceNoneByName = map[string]int64{"unknown": 0, "critical": 1}
var performanceNoneByID   = map[int64]string{0: "unknown", 1: "critical"}

// pop_type maps — email: "pop3"/"imap"/"pop3s"/"imaps" ↔ API int codes 110/143/995/993
var popTypeByName = map[string]int64{"pop3": 110, "imap": 143, "pop3s": 995, "imaps": 993}
var popTypeByID   = map[int64]string{110: "pop3", 143: "imap", 995: "pop3s", 993: "imaps"}

// delete_mode maps — email: "ok_only"/"all" ↔ API int codes 2/3
var deleteModeByName = map[string]int64{"ok_only": 2, "all": 3}
var deleteModeByID   = map[int64]string{2: "ok_only", 3: "all"}

// mongoConnectionByName — mongodb: "mongodb"/"mongodb_srv" ↔ API int codes 0/1
var mongoConnectionByName = map[string]int64{"mongodb": 0, "mongodb_srv": 1}
var mongoConnectionByID   = map[int64]string{0: "mongodb", 1: "mongodb_srv"}

// searchitem_none maps — local_web, system: "warning"/"critical"/"unknown" ↔ 1/2/3
var searchItemNoneByName = map[string]int64{"warning": 1, "critical": 2, "unknown": 3}
var searchItemNoneByID   = map[int64]string{1: "warning", 2: "critical", 3: "unknown"}

// portscan_none maps — port_scan: "critical"/"unknown" ↔ 2/3
var portscanNoneByName = map[string]int64{"critical": 2, "unknown": 3}
var portscanNoneByID   = map[int64]string{2: "critical", 3: "unknown"}

// sortedStringKeys returns the sorted keys of a map[string]T for use in error messages.
func sortedStringKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// checkTypeByDisplayName maps the display name returned by service_show ("name" field)
// to the internal type name used in Terraform config. Keys are lowercased for matching.
var checkTypeByDisplayName = map[string]string{
	"http":             "http",
	"http perf":        "http_perf",
	"site perf":        "site_perf",
	"ssl cert":         "ssl_cert",
	"domain":           "domain",
	"ping":             "ping",
	"ssh":              "ssh",
	"ftp":              "ftp",
	"tcp":              "tcp",
	"dns":              "dns",
	"smtp":             "smtp",
	"pop3":             "pop3",
	"imap":             "imap",
	"email":            "email",
	"blacklist":        "blacklist",
	"mysql":            "mysql",
	"postgresql":       "postgresql",
	"ms sql":           "mssql",
	"mongodb":          "mongodb",
	"local web":        "local_web",
	"local":            "local",
	"intranet":         "intranet",
	"cron":             "cron",
	"system":           "system",
	"transact server":  "transact_server",
	"transact browser": "transact_browser",
	"selenium":         "selenium",
	"postman":          "postman",
	"port scan":        "port_scan",
}

// requiredParamsByType maps each check type to the set of param attribute names
// that must be set for the API call to succeed. These are checked in ValidateConfig
// so the user gets a plan-time error instead of an opaque API 210 error at apply time.
var requiredParamsByType = map[string][]string{
	"http":             {"path", "timeout", "port"},
	"http_perf":        {"path", "warning_throughput", "warning_connect", "warning_roundtrip", "critical_throughput", "critical_connect", "critical_roundtrip", "timeout", "port"},
	"site_perf":        {"path", "warning_throughput", "warning_roundtrip", "critical_throughput", "critical_roundtrip", "timeout", "port"},
	"ssl_cert":         {"protocol", "port", "validity_warning", "validity_critical", "timeout"},
	"domain":           {"expected_ip1"},
	"ping":             {"response_warning", "response_critical", "timeout"},
	"ssh":              {"port", "timeout"},
	"ftp":              {"port", "timeout"},
	"tcp":              {"port", "timeout"},
	"dns":              {"dns_name", "timeout"},
	"smtp":             {"port", "timeout"},
	"pop3":             {"port", "timeout"},
	"imap":             {"port", "timeout"},
	"email":            {"smtp_port", "sender", "recipient", "pop_type", "pop_host", "pop_user", "pop_password", "delete_mode", "time_warning", "time_critical", "timeout"},
	"blacklist":        {},
	"mysql":            {"port", "username", "password", "timeout"},
	"postgresql":       {"port", "username", "password", "database", "timeout"},
	"mssql":            {"username", "password", "sql_query", "timeout", "port"},
	"mongodb":          {"mongo_connection", "port", "database", "username", "password", "timeout"},
	"local_web":        {"path", "searchitem_ok", "searchitem_critical", "timeout", "port", "searchitem_none"},
	"local":            {"port", "command", "timeout"},
	"intranet":         {"command"},
	"cron":             {"cron", "timezone", "timeout"},
	"system":           {"path", "port", "timeout", "searchitem_none"},
	"transact_server":  {"transaction_id"},
	"transact_browser": {"transaction_id"},
	"selenium":         {"definition", "language", "timeout"},
	"postman":          {"definition", "timeout"},
	"port_scan":        {"port_range", "portscan_none"},
}

// allowedParamsByType maps each check type to the set of param attribute names
// that are valid for that type, derived from api-docs/check-params.md.
var allowedParamsByType = map[string][]string{
	"http":             {"path", "timeout", "port", "searchitem", "method", "postvars", "regex", "invert_regex", "header_regex", "min_size", "max_size", "auth_type", "login", "password", "useragent", "addheader1", "addheader2", "addheader3", "content_type", "status_code", "redirect", "ssl_version", "check_cert"},
	"http_perf":        {"path", "timeout", "port", "warning_throughput", "warning_connect", "warning_roundtrip", "critical_throughput", "critical_connect", "critical_roundtrip", "useragent", "performance_none", "ssl_version", "host_header", "auth_type", "login", "password"},
	"site_perf":        {"path", "timeout", "port", "warning_throughput", "warning_roundtrip", "critical_throughput", "critical_roundtrip", "ssl", "useragent", "performance_none"},
	"ssl_cert":         {"protocol", "port", "validity_warning", "validity_critical", "timeout", "ssl_version", "check_cert", "different_ip", "addheader1"},
	"domain":           {"expected_ip1", "expected_ip2", "expected_ip3", "expected_ip4", "expected_iplink", "dns_server", "dns_record", "dns_auth", "timeout"},
	"ping":             {"response_warning", "response_critical", "timeout"},
	"ssh":              {"port", "timeout"},
	"ftp":              {"port", "timeout", "ssl"},
	"tcp":              {"port", "timeout"},
	"dns":              {"dns_name", "timeout"},
	"smtp":             {"port", "timeout", "from", "ssl_version"},
	"pop3":             {"port", "timeout", "ssl"},
	"imap":             {"port", "timeout", "ssl"},
	"email":            {"smtp_port", "sender", "recipient", "pop_type", "pop_host", "pop_user", "pop_password", "delete_mode", "time_warning", "time_critical", "timeout", "smtp_user", "smtp_password", "smtp_tls", "smtp_header1"},
	"blacklist":        {},
	"mysql":            {"port", "username", "password", "timeout"},
	"postgresql":       {"port", "username", "password", "database", "timeout"},
	"mssql":            {"port", "username", "password", "sql_query", "timeout"},
	"mongodb":          {"mongo_connection", "port", "database", "username", "password", "timeout"},
	"local_web":        {"path", "searchitem_ok", "searchitem_warning", "searchitem_critical", "searchitem_none", "timeout", "port", "ssl", "basicauth_login", "basicauth_password", "debug_addon", "performance", "performance_name", "performance_unit", "performance_digits", "host_header"},
	"local":            {"port", "command", "timeout", "ssl"},
	"intranet":         {"command", "token", "silent", "performance", "performance_name", "performance_regex", "performance_unit", "performance_digits"},
	"cron":             {"cron", "timezone", "timeout", "searchitem_ok", "searchitem_warning", "searchitem_critical", "performance", "performance_name", "performance_unit", "performance_digits"},
	"system":           {"path", "token", "port", "timeout", "searchitem_none", "ssl", "host_header", "def"},
	"transact_server":  {"transaction_id"},
	"transact_browser": {"transaction_id"},
	"selenium":         {"definition", "language", "timeout", "suite_filter"},
	"postman":          {"definition", "timeout"},
	"port_scan":        {"port_range", "portscan_none", "open_ports"},
}

// useStateForUnknown plan modifiers that also handle null state (unlike the
// built-in UseStateForUnknown which skips null state). This is needed so that
// Optional+Computed fields resolve to null (not "known after apply") after import
// when the API doesn't return a value for the field.

type useStateStringModifier struct{}

func (m useStateStringModifier) Description(_ context.Context) string {
	return "Use state value for unknown, including null."
}
func (m useStateStringModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (m useStateStringModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !resp.PlanValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}

type useStateBoolModifier struct{}

func (m useStateBoolModifier) Description(_ context.Context) string {
	return "Use state value for unknown, including null."
}
func (m useStateBoolModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (m useStateBoolModifier) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !resp.PlanValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}

type useStateInt64Modifier struct{}

func (m useStateInt64Modifier) Description(_ context.Context) string {
	return "Use state value for unknown, including null."
}
func (m useStateInt64Modifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (m useStateInt64Modifier) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if !resp.PlanValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}

type useStateFloat64Modifier struct{}

func (m useStateFloat64Modifier) Description(_ context.Context) string {
	return "Use state value for unknown, including null."
}
func (m useStateFloat64Modifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (m useStateFloat64Modifier) PlanModifyFloat64(_ context.Context, req planmodifier.Float64Request, resp *planmodifier.Float64Response) {
	if !resp.PlanValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}

type useStateObjectModifier struct{}

func (m useStateObjectModifier) Description(_ context.Context) string {
	return "Use state value for unknown, including null."
}
func (m useStateObjectModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (m useStateObjectModifier) PlanModifyObject(_ context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if !resp.PlanValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}

// validCheckTypes returns a sorted slice of all valid type names for use in
// error messages and validators.
func validCheckTypes() []string {
	names := make([]string, 0, len(checkTypeByName))
	for name := range checkTypeByName {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

// regionValidator validates that the region attribute is a known region name or
// a numeric string "1"–"5" (accepted for backwards compatibility).
type regionValidator struct{}

func (v regionValidator) Description(_ context.Context) string {
	return "must be a valid region name or numeric region ID"
}

func (v regionValidator) MarkdownDescription(_ context.Context) string {
	return "must be one of: `" + joinStrings(validRegions(), "`, `") + "` (or legacy numeric IDs 1–5)"
}

func (v regionValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if _, ok := regionByName[val]; ok {
		return
	}
	// Accept numeric string IDs for backwards compatibility.
	if id, err := strconv.ParseInt(val, 10, 64); err == nil {
		if _, ok := regionByID[id]; ok {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid region",
		fmt.Sprintf("%q is not a valid region. Valid names: %s (or numeric IDs 0–5)",
			val,
			joinStrings(validRegions(), ", "),
		),
	)
}

// checkTypeValidator validates that the type attribute is a known check type name.
type checkTypeValidator struct{}

func (v checkTypeValidator) Description(_ context.Context) string {
	return "must be a valid check type name"
}

func (v checkTypeValidator) MarkdownDescription(_ context.Context) string {
	return "must be one of: `" + joinStrings(validCheckTypes(), "`, `") + "`"
}

func (v checkTypeValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, ok := checkTypeByName[req.ConfigValue.ValueString()]; !ok {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid check type",
			fmt.Sprintf("%q is not a valid check type. Valid values: %s",
				req.ConfigValue.ValueString(),
				joinStrings(validCheckTypes(), ", "),
			),
		)
	}
}

var _ resource.Resource = &CheckResource{}
var _ resource.ResourceWithImportState = &CheckResource{}
var _ resource.ResourceWithValidateConfig = &CheckResource{}
var _ resource.ResourceWithModifyPlan = &CheckResource{}

type CheckResource struct {
	client *client.Client
}

// CheckParamsModel holds type-specific check parameters.
type CheckParamsModel struct {
	// Common check params
	Path               types.String  `tfsdk:"path"`
	Timeout            types.Int64   `tfsdk:"timeout"`
	Port               types.Int64   `tfsdk:"port"`
	Ssl                types.Bool    `tfsdk:"ssl"`
	SslVersion         types.String  `tfsdk:"ssl_version"`
	CheckCert          types.Bool    `tfsdk:"check_cert"`
	AuthType           types.String  `tfsdk:"auth_type"`
	Login              types.String  `tfsdk:"login"`
	Password           types.String  `tfsdk:"password"`
	Username           types.String  `tfsdk:"username"`
	Useragent          types.String  `tfsdk:"useragent"`
	HostHeader         types.String  `tfsdk:"host_header"`
	// HTTP params
	Searchitem         types.String  `tfsdk:"searchitem"`
	Method             types.String  `tfsdk:"method"`
	Postvars           types.String  `tfsdk:"postvars"`
	Regex              types.String  `tfsdk:"regex"`
	InvertRegex        types.Bool    `tfsdk:"invert_regex"`
	HeaderRegex        types.String  `tfsdk:"header_regex"`
	MinSize            types.Int64   `tfsdk:"min_size"`
	MaxSize            types.Int64   `tfsdk:"max_size"`
	Addheader1         types.String  `tfsdk:"addheader1"`
	Addheader2         types.String  `tfsdk:"addheader2"`
	Addheader3         types.String  `tfsdk:"addheader3"`
	ContentType        types.String  `tfsdk:"content_type"`
	StatusCode         types.Int64   `tfsdk:"status_code"`
	Redirect           types.Bool    `tfsdk:"redirect"`
	// Perf params
	WarningThroughput  types.Float64 `tfsdk:"warning_throughput"`
	WarningConnect     types.Float64 `tfsdk:"warning_connect"`
	WarningRoundtrip   types.Float64 `tfsdk:"warning_roundtrip"`
	CriticalThroughput types.Float64 `tfsdk:"critical_throughput"`
	CriticalConnect    types.Float64 `tfsdk:"critical_connect"`
	CriticalRoundtrip  types.Float64 `tfsdk:"critical_roundtrip"`
	PerformanceNone    types.String  `tfsdk:"performance_none"`
	// SSL Cert params
	Protocol           types.String  `tfsdk:"protocol"`
	ValidityWarning    types.Int64   `tfsdk:"validity_warning"`
	ValidityCritical   types.Int64   `tfsdk:"validity_critical"`
	DifferentIP        types.String  `tfsdk:"different_ip"`
	// Domain params
	ExpectedIP1        types.String  `tfsdk:"expected_ip1"`
	ExpectedIP2        types.String  `tfsdk:"expected_ip2"`
	ExpectedIP3        types.String  `tfsdk:"expected_ip3"`
	ExpectedIP4        types.String  `tfsdk:"expected_ip4"`
	ExpectedIPLink     types.String  `tfsdk:"expected_iplink"`
	DnsServer          types.String  `tfsdk:"dns_server"`
	DnsRecord          types.String  `tfsdk:"dns_record"`
	DnsAuth            types.String  `tfsdk:"dns_auth"`
	// PING params
	ResponseWarning    types.Int64   `tfsdk:"response_warning"`
	ResponseCritical   types.Int64   `tfsdk:"response_critical"`
	// DNS params
	DNSName            types.String  `tfsdk:"dns_name"`
	// SMTP params
	From               types.String  `tfsdk:"from"`
	// Email params
	SmtpPort           types.Int64   `tfsdk:"smtp_port"`
	Sender             types.String  `tfsdk:"sender"`
	Recipient          types.String  `tfsdk:"recipient"`
	PopType            types.String  `tfsdk:"pop_type"`
	PopHost            types.String  `tfsdk:"pop_host"`
	PopUser            types.String  `tfsdk:"pop_user"`
	PopPassword        types.String  `tfsdk:"pop_password"`
	DeleteMode         types.String  `tfsdk:"delete_mode"`
	TimeWarning        types.Int64   `tfsdk:"time_warning"`
	TimeCritical       types.Int64   `tfsdk:"time_critical"`
	SmtpUser           types.String  `tfsdk:"smtp_user"`
	SmtpPassword       types.String  `tfsdk:"smtp_password"`
	SmtpTls            types.Bool    `tfsdk:"smtp_tls"`
	SmtpHeader1        types.String  `tfsdk:"smtp_header1"`
	// Database params
	Database           types.String  `tfsdk:"database"`
	SQLQuery           types.String  `tfsdk:"sql_query"`
	MongoConnection    types.String  `tfsdk:"mongo_connection"`
	// Local Web / Search params
	SearchItemOk       types.String  `tfsdk:"searchitem_ok"`
	SearchItemWarning  types.String  `tfsdk:"searchitem_warning"`
	SearchItemCritical types.String  `tfsdk:"searchitem_critical"`
	SearchItemNone     types.String  `tfsdk:"searchitem_none"`
	BasicauthLogin     types.String  `tfsdk:"basicauth_login"`
	BasicauthPassword  types.String  `tfsdk:"basicauth_password"`
	DebugAddon         types.String  `tfsdk:"debug_addon"`
	// Local / Intranet params
	Command            types.String  `tfsdk:"command"`
	Token              types.String  `tfsdk:"token"`
	Silent             types.Int64   `tfsdk:"silent"`
	// Cron params
	Cron               types.String  `tfsdk:"cron"`
	Timezone           types.String  `tfsdk:"timezone"`
	// System params
	Def                types.String  `tfsdk:"def"`
	// Performance params
	Performance        types.Bool    `tfsdk:"performance"`
	PerformanceName    types.String  `tfsdk:"performance_name"`
	PerformanceRegex   types.String  `tfsdk:"performance_regex"`
	PerformanceUnit    types.String  `tfsdk:"performance_unit"`
	PerformanceDigits  types.Int64   `tfsdk:"performance_digits"`
	// Transaction params
	TransactionID      types.String  `tfsdk:"transaction_id"`
	// Selenium / Postman params
	Definition         types.String  `tfsdk:"definition"`
	Language           types.String  `tfsdk:"language"`
	SuiteFilter        types.String  `tfsdk:"suite_filter"`
	// Port Scan params
	PortRange          types.String  `tfsdk:"port_range"`
	PortscanNone       types.String  `tfsdk:"portscan_none"`
	OpenPorts          types.String  `tfsdk:"open_ports"`
}

// CheckResourceModel holds all attributes for the check resource.
type CheckResourceModel struct {
	ID           types.String `tfsdk:"id"`
	HostID       types.String `tfsdk:"host_id"`
	Type         types.String `tfsdk:"type"`
	Shortname    types.String `tfsdk:"shortname"`
	Description  types.String `tfsdk:"description"`
	Address      types.String `tfsdk:"address"`
	Region       types.String `tfsdk:"region"`
	IPv6         types.Bool   `tfsdk:"ipv6"`
	Interval     types.Float64 `tfsdk:"interval"`
	NotifyAfter  types.Float64 `tfsdk:"notify_after"`
	Reminder     types.Float64 `tfsdk:"reminder_interval"`
	Message      types.String  `tfsdk:"message"`
	Active       types.Bool    `tfsdk:"active"`
	Notification types.Bool    `tfsdk:"notification"`

	// Notification flags
	EmailCritical types.Bool `tfsdk:"email_critical"`
	EmailWarning  types.Bool `tfsdk:"email_warning"`
	EmailOk       types.Bool `tfsdk:"email_ok"`
	EmailRemind   types.Bool `tfsdk:"email_remind"`
	SmsCritical   types.Bool `tfsdk:"sms_critical"`
	SmsWarning    types.Bool `tfsdk:"sms_warning"`
	SmsOk         types.Bool `tfsdk:"sms_ok"`
	SmsRemind     types.Bool `tfsdk:"sms_remind"`
	PushCritical  types.Bool `tfsdk:"push_critical"`
	PushWarning   types.Bool `tfsdk:"push_warning"`
	PushOk        types.Bool `tfsdk:"push_ok"`
	PushRemind    types.Bool `tfsdk:"push_remind"`
	HttpCritical  types.Bool `tfsdk:"http_critical"`
	HttpWarning   types.Bool `tfsdk:"http_warning"`
	HttpOk        types.Bool `tfsdk:"http_ok"`
	HttpRemind    types.Bool `tfsdk:"http_remind"`
	PhoneCritical types.Bool `tfsdk:"phone_critical"`
	PhoneWarning  types.Bool `tfsdk:"phone_warning"`
	PhoneOk       types.Bool `tfsdk:"phone_ok"`
	PhoneRemind   types.Bool `tfsdk:"phone_remind"`

	// Type-specific parameters
	Params *CheckParamsModel `tfsdk:"params"`
}

func NewCheckResource() resource.Resource {
	return &CheckResource{}
}

func (r *CheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check"
}

func (r *CheckResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	boolOpt := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc,
			PlanModifiers: []planmodifier.Bool{useStateBoolModifier{}}}
	}
	strOpt := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc,
			PlanModifiers: []planmodifier.String{useStateStringModifier{}}}
	}
	intOpt := func(desc string) schema.Int64Attribute {
		return schema.Int64Attribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc,
			PlanModifiers: []planmodifier.Int64{useStateInt64Modifier{}}}
	}
	floatOpt := func(desc string) schema.Float64Attribute {
		return schema.Float64Attribute{Optional: true, Computed: true, Description: desc, MarkdownDescription: desc,
			PlanModifiers: []planmodifier.Float64{useStateFloat64Modifier{}}}
	}
	sensitiveStrOpt := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Sensitive: true, Description: desc, MarkdownDescription: desc,
			PlanModifiers: []planmodifier.String{useStateStringModifier{}}}
	}

	resp.Schema = schema.Schema{
		Description:         "Manages a Serverguard24 monitoring check.",
		MarkdownDescription: "Manages a Serverguard24 monitoring check.",
		Version:             1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"host_id": schema.StringAttribute{Required: true, Description: "ID of the host this check belongs to.", MarkdownDescription: "ID of the host this check belongs to."},
			"type": schema.StringAttribute{
				Required:            true,
				Description:         "Check type name (e.g. http, ping, ssl_cert).",
				MarkdownDescription: "Check type. Valid values: `" + joinStrings(validCheckTypes(), "`, `") + "`.",
				Validators:          []validator.String{checkTypeValidator{}},
			},
			"shortname":         strOpt("Short display name (max 40 characters)."),
			"description":       strOpt("Description of this check."),
			"address":           strOpt("Override address for this check."),
			"region": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Monitoring region. Valid values: individual, europe, americas, europe_americas, asia_pacific, world. Use \"individual\" to enable per-node selection (configure nodes in the Serverguard24 UI). Numeric IDs (0–5) are accepted for backwards compatibility and normalized to names on read.",
				MarkdownDescription: "Monitoring region. Valid values: `individual`, `europe`, `americas`, `europe_americas`, `asia_pacific`, `world`. Use `\"individual\"` to enable per-node selection (configure nodes in the Serverguard24 UI). Numeric IDs `0`–`5` are accepted for backwards compatibility.",
				Validators:          []validator.String{regionValidator{}},
				PlanModifiers:       []planmodifier.String{useStateStringModifier{}},
			},
			"ipv6":              boolOpt("Use IPv6."),
			"interval":          floatOpt("Check interval in minutes."),
			"notify_after":      floatOpt("Notify after N minutes of downtime."),
			"reminder_interval": floatOpt("Reminder interval in hours."),
			"message":           strOpt("Custom notification message."),
			"active": schema.BoolAttribute{
				Optional: true, Computed: true,
				Default:             booldefault.StaticBool(true),
				Description:         "Set to false to deactivate the check without deleting it. Defaults to true.",
				MarkdownDescription: "Set to `false` to deactivate the check without deleting it. Defaults to `true`.",
			},
			"notification": schema.BoolAttribute{
				Optional: true, Computed: true,
				Default:             booldefault.StaticBool(true),
				Description:         "Enable or disable alert notifications for this check. Defaults to true.",
				MarkdownDescription: "Enable or disable alert notifications for this check. Defaults to `true`.",
			},
			// Notification flags
			"email_critical": boolOpt("Send email on critical state."),
			"email_warning":  boolOpt("Send email on warning state."),
			"email_ok":       boolOpt("Send email on recovery."),
			"email_remind":   boolOpt("Send email reminders."),
			"sms_critical":   boolOpt("Send SMS on critical state."),
			"sms_warning":    boolOpt("Send SMS on warning state."),
			"sms_ok":         boolOpt("Send SMS on recovery."),
			"sms_remind":     boolOpt("Send SMS reminders."),
			"push_critical":  boolOpt("Send push on critical state."),
			"push_warning":   boolOpt("Send push on warning state."),
			"push_ok":        boolOpt("Send push on recovery."),
			"push_remind":    boolOpt("Send push reminders."),
			"http_critical":  boolOpt("Send HTTP hook on critical state."),
			"http_warning":   boolOpt("Send HTTP hook on warning state."),
			"http_ok":        boolOpt("Send HTTP hook on recovery."),
			"http_remind":    boolOpt("Send HTTP hook reminders."),
			"phone_critical": boolOpt("Send phone call on critical state."),
			"phone_warning":  boolOpt("Send phone call on warning state."),
			"phone_ok":       boolOpt("Send phone call on recovery."),
			"phone_remind":   boolOpt("Send phone call reminders."),
		},
		Blocks: map[string]schema.Block{
			"params": schema.SingleNestedBlock{
				Description:         "Type-specific parameters. Valid attributes depend on the check type.",
				MarkdownDescription: "Type-specific parameters. Valid attributes depend on the check type.",
				PlanModifiers:       []planmodifier.Object{useStateObjectModifier{}},
				Attributes: map[string]schema.Attribute{
					// Common
					"path":               strOpt("URL path."),
					"timeout":            intOpt("Timeout in seconds."),
					"port":               intOpt("Port number."),
					"ssl":                boolOpt("Enable SSL/TLS encryption."),
					"ssl_version":        strOpt(`SSL/TLS protocol version. Valid values depend on check type — http/http_perf: "auto", "tlsv1", "sslv3"; ssl_cert: "auto", "tlsv1", "sslv3"; smtp: "none", "tls", "ssl".`),
					"check_cert":         boolOpt("Verify SSL certificate."),
					"auth_type":          strOpt(`Authentication method: "none", "basic", "ntlm".`),
					"login":              strOpt("Login username for authentication."),
					"password":           sensitiveStrOpt("Password."),
					"username":           strOpt("Username (database checks)."),
					"useragent":          strOpt("User agent string."),
					"host_header":        strOpt("Alternative host header for load balancers."),
					// HTTP
					"searchitem":         strOpt("Search term that must be found on the page."),
					"method":             strOpt("HTTP method (GET, POST, PUT, PATCH, DELETE)."),
					"postvars":           strOpt("POST variables (field1=val1&field2=val2)."),
					"regex":              strOpt("Search term as regular expression."),
					"invert_regex":       boolOpt("Negate regex match."),
					"header_regex":       strOpt("Regex matched against response headers."),
					"min_size":           intOpt("Minimum page size in bytes."),
					"max_size":           intOpt("Maximum page size in bytes."),
					"addheader1":         strOpt("Additional HTTP header (Name: Value)."),
					"addheader2":         strOpt("Additional HTTP header."),
					"addheader3":         strOpt("Additional HTTP header."),
					"content_type":       strOpt("Content-Type for the request."),
					"status_code":        intOpt("Expected HTTP status code (0=ignore, default: 200)."),
					"redirect":           boolOpt("Follow HTTP redirects (301/302)."),
					// Perf
					"warning_throughput":  floatOpt("Warning threshold for throughput (KB/s)."),
					"warning_connect":     floatOpt("Warning threshold for connect time (ms)."),
					"warning_roundtrip":   floatOpt("Warning threshold for roundtrip time (ms)."),
					"critical_throughput": floatOpt("Critical threshold for throughput (KB/s)."),
					"critical_connect":    floatOpt("Critical threshold for connect time (ms)."),
					"critical_roundtrip":  floatOpt("Critical threshold for roundtrip time (ms)."),
					"performance_none":    strOpt(`Status when no performance value can be measured: "unknown", "critical".`),
					// SSL Cert
					"protocol":           strOpt("Protocol (HTTPS, SMTPS, IMAPS, POP3S)."),
					"validity_warning":   intOpt("Warn when cert expires within N days."),
					"validity_critical":  intOpt("Alert when cert expires within N days."),
					"different_ip":       strOpt("Alternative node IP for load balancers."),
					// Domain
					"expected_ip1":       strOpt("Expected IP address or domain."),
					"expected_ip2":       strOpt("Additional expected address."),
					"expected_ip3":       strOpt("Additional expected address."),
					"expected_ip4":       strOpt("Additional expected address."),
					"expected_iplink":    strOpt("Address link logic (OR, AND, ALL)."),
					"dns_server":         strOpt("DNS server to use."),
					"dns_record":         strOpt("DNS record type (A, AAAA, CNAME, MX)."),
					"dns_auth":           strOpt("DNS server must be authoritative."),
					// PING
					"response_warning":   intOpt("Warning response time (ms)."),
					"response_critical":  intOpt("Critical response time (ms)."),
					// DNS
					"dns_name":           strOpt("DNS name to resolve."),
					// SMTP
					"from":               strOpt("From address for SMTP MAILTO command."),
					// Email
					"smtp_port":          intOpt("SMTP port for email checks."),
					"sender":             strOpt("Sender email address."),
					"recipient":          strOpt("Recipient email address."),
					"pop_type":           strOpt(`Recipient mailbox type: "pop3", "imap", "pop3s", "imaps".`),
					"pop_host":           strOpt("Mailbox server address."),
					"pop_user":           strOpt("POP/IMAP username."),
					"pop_password":       sensitiveStrOpt("POP/IMAP password."),
					"delete_mode":        strOpt(`Email deletion mode: "ok_only" (delete only test emails that arrived OK), "all" (delete all emails).`),
					"time_warning":       intOpt("Warning delivery time (minutes)."),
					"time_critical":      intOpt("Critical delivery time (minutes)."),
					"smtp_user":          strOpt("SMTP username."),
					"smtp_password":      sensitiveStrOpt("SMTP password."),
					"smtp_tls":           boolOpt("SMTP encryption."),
					"smtp_header1":       strOpt("Additional SMTP header."),
					// Database
					"database":           strOpt("Database name."),
					"sql_query":          strOpt("SQL query to execute."),
					"mongo_connection":   strOpt(`MongoDB connection type: "mongodb" (standard URI) or "mongodb_srv" (SRV URI).`),
					// Local Web / Search
					"searchitem_ok":       strOpt("Search term indicating OK status."),
					"searchitem_warning":  strOpt("Search term indicating Warning status."),
					"searchitem_critical": strOpt("Search term indicating Critical status."),
					"searchitem_none":     strOpt(`Status when no search term is found: "warning", "critical", "unknown".`),
					"basicauth_login":     strOpt("Basic Authentication username."),
					"basicauth_password":  sensitiveStrOpt("Basic Authentication password."),
					"debug_addon":         strOpt("URL debug suffix."),
					// Local / Intranet
					"command":            strOpt("Command to run (local/intranet checks). Sent as 'cmd' to the API."),
					"token":              sensitiveStrOpt("Check token (e.g. intranet/NCPA). Generated by Serverguard24 when omitted."),
					"silent":             intOpt("Warning after x minutes of silence (0=disabled)."),
					// Cron
					"cron":               strOpt("Cron expression."),
					"timezone":           strOpt("Timezone for cron checks."),
					// System
					"def":                strOpt("System check definition."),
					// Performance
					"performance":        boolOpt("Check returns performance data."),
					"performance_name":   strOpt("Performance value label."),
					"performance_regex":  strOpt("Regex to extract performance value."),
					"performance_unit":   strOpt("Performance value unit."),
					"performance_digits": intOpt("Performance value decimal places (0-5)."),
					// Transaction
					"transaction_id":     strOpt("Transaction ID for TransAct checks."),
					// Selenium / Postman
					"definition":         strOpt("Script definition (Selenium/Postman)."),
					"language":           strOpt("Script language (Selenium)."),
					"suite_filter":       strOpt("Test suite filter (Selenium)."),
					// Port Scan
					"port_range":         strOpt("Port range to scan."),
					"portscan_none":      strOpt(`Status when port scan is not possible: "critical", "unknown".`),
					"open_ports":         strOpt("Allowed open ports."),
				},
			},
		},
	}
}

func (r *CheckResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.client = c
}

// ValidateConfig checks that only params valid for the selected check type are set.
func (r *CheckResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config CheckResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Type.IsNull() || config.Type.IsUnknown() {
		return
	}
	typeName := config.Type.ValueString()

	allowed, ok := allowedParamsByType[typeName]
	if !ok {
		return // type validator catches invalid types
	}

	if config.Params == nil {
		return
	}

	allowedSet := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = true
	}

	// Check each non-null param against the allowed set
	for _, pc := range paramsSetFields(config.Params) {
		if pc.isSet && !allowedSet[pc.name] {
			resp.Diagnostics.AddAttributeError(
				path.Root("params").AtName(pc.name),
				"Invalid parameter for check type",
				fmt.Sprintf("Parameter %q is not valid for check type %q. Valid parameters: %s",
					pc.name, typeName, strings.Join(allowed, ", ")),
			)
		}
	}

	// Check that all required params for this type are present.
	// Unknown values (e.g. from for_each expressions) are treated as potentially
	// present — validation is deferred to apply time for those.
	required := requiredParamsByType[typeName]
	if len(required) > 0 {
		setMap := make(map[string]bool, len(paramsSetFields(config.Params)))
		for _, pc := range paramsSetFields(config.Params) {
			if pc.isSet || pc.unknown {
				setMap[pc.name] = true
			}
		}
		for _, req := range required {
			if !setMap[req] {
				resp.Diagnostics.AddAttributeError(
					path.Root("params").AtName(req),
					"Missing required parameter",
					fmt.Sprintf("Parameter %q is required for check type %q.", req, typeName),
				)
			}
		}
	}

	// Validate ssl_version string value against the type-specific allowed set.
	if p := config.Params; p != nil && allowedSet["ssl_version"] &&
		!p.SslVersion.IsNull() && !p.SslVersion.IsUnknown() {
		nameToID, _ := sslVersionMaps(typeName)
		if _, ok := nameToID[p.SslVersion.ValueString()]; !ok {
			resp.Diagnostics.AddAttributeError(
				path.Root("params").AtName("ssl_version"),
				"Invalid ssl_version",
				fmt.Sprintf("ssl_version %q is not valid for check type %q. Valid values: %s",
					p.SslVersion.ValueString(), typeName,
					strings.Join(validSSLVersions(typeName), ", ")),
			)
		}
	}

	validateEnumParam := func(attrName string, value types.String, valid map[string]string) {
		if config.Params == nil || !allowedSet[attrName] || value.IsNull() || value.IsUnknown() {
			return
		}
		if _, ok := valid[value.ValueString()]; !ok {
			resp.Diagnostics.AddAttributeError(
				path.Root("params").AtName(attrName),
				fmt.Sprintf("Invalid %s", attrName),
				fmt.Sprintf("%q is not valid. Allowed: %s", value.ValueString(),
					strings.Join(sortedStringKeys(valid), ", ")),
			)
		}
	}
	validateEnumParamInt := func(attrName string, value types.String, valid map[string]int64) {
		if config.Params == nil || !allowedSet[attrName] || value.IsNull() || value.IsUnknown() {
			return
		}
		if _, ok := valid[value.ValueString()]; !ok {
			resp.Diagnostics.AddAttributeError(
				path.Root("params").AtName(attrName),
				fmt.Sprintf("Invalid %s", attrName),
				fmt.Sprintf("%q is not valid. Allowed: %s", value.ValueString(),
					strings.Join(sortedStringKeys(valid), ", ")),
			)
		}
	}

	if p := config.Params; p != nil {
		validateEnumParam("auth_type", p.AuthType, authTypeByName)
		validateEnumParamInt("performance_none", p.PerformanceNone, performanceNoneByName)
		validateEnumParamInt("pop_type", p.PopType, popTypeByName)
		validateEnumParamInt("delete_mode", p.DeleteMode, deleteModeByName)
		validateEnumParamInt("mongo_connection", p.MongoConnection, mongoConnectionByName)
		validateEnumParamInt("searchitem_none", p.SearchItemNone, searchItemNoneByName)
		validateEnumParamInt("portscan_none", p.PortscanNone, portscanNoneByName)
	}
}

// ModifyPlan warns when the user tries to clear an optional string or toggle a
// notification flag off. The Serverguard24 service_edit API treats both cases
// as no-ops, producing a perpetual diff. The active/notification fields are
// handled by separate API calls (service_monitoring/service_notification) and
// do not have this limitation.
func (r *CheckResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var state, plan CheckResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	warnIfClearingString(&resp.Diagnostics, path.Root("shortname"), state.Shortname, plan.Shortname)
	warnIfClearingString(&resp.Diagnostics, path.Root("description"), state.Description, plan.Description)
	warnIfClearingString(&resp.Diagnostics, path.Root("address"), state.Address, plan.Address)
	warnIfClearingString(&resp.Diagnostics, path.Root("message"), state.Message, plan.Message)

	warnIfClearingBool(&resp.Diagnostics, path.Root("ipv6"), state.IPv6, plan.IPv6)

	type boolPair struct {
		name        string
		state, plan types.Bool
	}
	notificationPairs := []boolPair{
		{"email_critical", state.EmailCritical, plan.EmailCritical},
		{"email_warning", state.EmailWarning, plan.EmailWarning},
		{"email_ok", state.EmailOk, plan.EmailOk},
		{"email_remind", state.EmailRemind, plan.EmailRemind},
		{"sms_critical", state.SmsCritical, plan.SmsCritical},
		{"sms_warning", state.SmsWarning, plan.SmsWarning},
		{"sms_ok", state.SmsOk, plan.SmsOk},
		{"sms_remind", state.SmsRemind, plan.SmsRemind},
		{"push_critical", state.PushCritical, plan.PushCritical},
		{"push_warning", state.PushWarning, plan.PushWarning},
		{"push_ok", state.PushOk, plan.PushOk},
		{"push_remind", state.PushRemind, plan.PushRemind},
		{"http_critical", state.HttpCritical, plan.HttpCritical},
		{"http_warning", state.HttpWarning, plan.HttpWarning},
		{"http_ok", state.HttpOk, plan.HttpOk},
		{"http_remind", state.HttpRemind, plan.HttpRemind},
		{"phone_critical", state.PhoneCritical, plan.PhoneCritical},
		{"phone_warning", state.PhoneWarning, plan.PhoneWarning},
		{"phone_ok", state.PhoneOk, plan.PhoneOk},
		{"phone_remind", state.PhoneRemind, plan.PhoneRemind},
	}
	for _, pp := range notificationPairs {
		warnIfClearingBool(&resp.Diagnostics, path.Root(pp.name), pp.state, pp.plan)
	}

	// Params block: warn for the bool flags and a handful of common strings that
	// users commonly edit. The full param list is too large to enumerate here;
	// see the provider docs for the general caveat.
	if state.Params != nil && plan.Params != nil {
		sp, pp := state.Params, plan.Params
		warnIfClearingBool(&resp.Diagnostics, path.Root("params").AtName("ssl"), sp.Ssl, pp.Ssl)
		warnIfClearingBool(&resp.Diagnostics, path.Root("params").AtName("check_cert"), sp.CheckCert, pp.CheckCert)
		warnIfClearingBool(&resp.Diagnostics, path.Root("params").AtName("invert_regex"), sp.InvertRegex, pp.InvertRegex)
		warnIfClearingBool(&resp.Diagnostics, path.Root("params").AtName("redirect"), sp.Redirect, pp.Redirect)
		warnIfClearingBool(&resp.Diagnostics, path.Root("params").AtName("smtp_tls"), sp.SmtpTls, pp.SmtpTls)
		warnIfClearingBool(&resp.Diagnostics, path.Root("params").AtName("performance"), sp.Performance, pp.Performance)
	}
}

type paramCheck struct {
	name    string
	isSet   bool
	unknown bool
}

func isSet(v interface{ IsNull() bool; IsUnknown() bool }) bool {
	return !v.IsNull() && !v.IsUnknown()
}

func paramField(name string, v interface{ IsNull() bool; IsUnknown() bool }) paramCheck {
	return paramCheck{name: name, isSet: isSet(v), unknown: v.IsUnknown()}
}

func paramsSetFields(p *CheckParamsModel) []paramCheck {
	return []paramCheck{
		paramField("path", p.Path), paramField("timeout", p.Timeout), paramField("port", p.Port),
		paramField("ssl", p.Ssl), paramField("ssl_version", p.SslVersion), paramField("check_cert", p.CheckCert),
		paramField("auth_type", p.AuthType), paramField("login", p.Login), paramField("password", p.Password),
		paramField("username", p.Username), paramField("useragent", p.Useragent), paramField("host_header", p.HostHeader),
		paramField("searchitem", p.Searchitem), paramField("method", p.Method), paramField("postvars", p.Postvars),
		paramField("regex", p.Regex), paramField("invert_regex", p.InvertRegex), paramField("header_regex", p.HeaderRegex),
		paramField("min_size", p.MinSize), paramField("max_size", p.MaxSize),
		paramField("addheader1", p.Addheader1), paramField("addheader2", p.Addheader2), paramField("addheader3", p.Addheader3),
		paramField("content_type", p.ContentType), paramField("status_code", p.StatusCode), paramField("redirect", p.Redirect),
		paramField("warning_throughput", p.WarningThroughput), paramField("warning_connect", p.WarningConnect),
		paramField("warning_roundtrip", p.WarningRoundtrip), paramField("critical_throughput", p.CriticalThroughput),
		paramField("critical_connect", p.CriticalConnect), paramField("critical_roundtrip", p.CriticalRoundtrip),
		paramField("performance_none", p.PerformanceNone),
		paramField("protocol", p.Protocol), paramField("validity_warning", p.ValidityWarning),
		paramField("validity_critical", p.ValidityCritical), paramField("different_ip", p.DifferentIP),
		paramField("expected_ip1", p.ExpectedIP1), paramField("expected_ip2", p.ExpectedIP2),
		paramField("expected_ip3", p.ExpectedIP3), paramField("expected_ip4", p.ExpectedIP4),
		paramField("expected_iplink", p.ExpectedIPLink), paramField("dns_server", p.DnsServer),
		paramField("dns_record", p.DnsRecord), paramField("dns_auth", p.DnsAuth),
		paramField("response_warning", p.ResponseWarning), paramField("response_critical", p.ResponseCritical),
		paramField("dns_name", p.DNSName), paramField("from", p.From),
		paramField("smtp_port", p.SmtpPort), paramField("sender", p.Sender), paramField("recipient", p.Recipient),
		paramField("pop_type", p.PopType), paramField("pop_host", p.PopHost), paramField("pop_user", p.PopUser),
		paramField("pop_password", p.PopPassword), paramField("delete_mode", p.DeleteMode),
		paramField("time_warning", p.TimeWarning), paramField("time_critical", p.TimeCritical),
		paramField("smtp_user", p.SmtpUser), paramField("smtp_password", p.SmtpPassword),
		paramField("smtp_tls", p.SmtpTls), paramField("smtp_header1", p.SmtpHeader1),
		paramField("database", p.Database), paramField("sql_query", p.SQLQuery), paramField("mongo_connection", p.MongoConnection),
		paramField("searchitem_ok", p.SearchItemOk), paramField("searchitem_warning", p.SearchItemWarning),
		paramField("searchitem_critical", p.SearchItemCritical), paramField("searchitem_none", p.SearchItemNone),
		paramField("basicauth_login", p.BasicauthLogin), paramField("basicauth_password", p.BasicauthPassword),
		paramField("debug_addon", p.DebugAddon),
		paramField("command", p.Command), paramField("token", p.Token), paramField("silent", p.Silent),
		paramField("cron", p.Cron), paramField("timezone", p.Timezone), paramField("def", p.Def),
		paramField("performance", p.Performance), paramField("performance_name", p.PerformanceName),
		paramField("performance_regex", p.PerformanceRegex), paramField("performance_unit", p.PerformanceUnit),
		paramField("performance_digits", p.PerformanceDigits),
		paramField("transaction_id", p.TransactionID), paramField("definition", p.Definition),
		paramField("language", p.Language), paramField("suite_filter", p.SuiteFilter),
		paramField("port_range", p.PortRange), paramField("portscan_none", p.PortscanNone), paramField("open_ports", p.OpenPorts),
	}
}

func (r *CheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CheckResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := buildCheckCommand("service_add", "", plan)
	apiResp, err := r.client.Do(ctx, cmd)
	if err != nil {
		resp.Diagnostics.AddError("Error creating check", err.Error())
		return
	}

	var data struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		resp.Diagnostics.AddError("Error parsing check ID", err.Error())
		return
	}

	plan.ID = types.StringValue(data.ID.String())

	if err := r.applyMonitoring(ctx, plan.ID.ValueString(), plan.Active); err != nil {
		resp.Diagnostics.AddError("Error setting active state", err.Error())
		return
	}
	if err := r.applyNotification(ctx, plan.ID.ValueString(), plan.Notification); err != nil {
		resp.Diagnostics.AddError("Error setting notification state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// refreshFromAPI calls service_show and merges the API response into state.
// Returns false if the resource no longer exists (caller should remove from state).
// host_id is preserved from the existing state since service_show does not return it.
func (r *CheckResource) refreshFromAPI(ctx context.Context, id string, state *CheckResourceModel) (found bool, err error) {
	apiResp, err := r.client.Do(ctx, client.Request{
		"command": "service_show",
		"id":      id,
	})
	if err != nil {
		if client.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	// service_show returns a single object. Flatten nested sub-objects so that
	// applyCheckData can access all fields at the top level.
	var raw map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &raw); err != nil {
		return false, err
	}
	if params, ok := raw["params"].(map[string]interface{}); ok {
		for k, v := range params {
			raw[k] = v
		}
	}
	if notif, ok := raw["notification"].(map[string]interface{}); ok {
		for k, v := range notif {
			raw[k] = v
		}
	}

	applyCheckData(state, raw)
	return true, nil
}

func (r *CheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CheckResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.refreshFromAPI(ctx, state.ID.ValueString(), &state)
	if err != nil {
		resp.Diagnostics.AddError("Error reading check", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CheckResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CheckResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := buildCheckCommand("service_edit", state.ID.ValueString(), plan)
	if _, err := r.client.Do(ctx, cmd); err != nil {
		resp.Diagnostics.AddError("Error updating check", err.Error())
		return
	}

	plan.ID = state.ID

	// service_monitoring / service_notification are only re-issued when the
	// corresponding attribute actually changes. The API treats them as toggles,
	// so a no-op edit is wasted load on the rate limiter.
	if !plan.Active.Equal(state.Active) {
		if err := r.applyMonitoring(ctx, plan.ID.ValueString(), plan.Active); err != nil {
			resp.Diagnostics.AddError("Error setting active state", err.Error())
			return
		}
	}
	if !plan.Notification.Equal(state.Notification) {
		if err := r.applyNotification(ctx, plan.ID.ValueString(), plan.Notification); err != nil {
			resp.Diagnostics.AddError("Error setting notification state", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CheckResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Do(ctx, client.Request{
		"command": "service_delete",
		"id":      state.ID.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error deleting check", err.Error())
	}
}

func (r *CheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, CheckResourceModel{
		ID: types.StringValue(req.ID),
	})...)
}

func buildCheckCommand(command, id string, m CheckResourceModel) client.Request {
	cmd := client.Request{
		"command": command,
	}
	if id != "" {
		cmd["id"] = id
	} else {
		// host_id and type are only required for service_add, not service_edit
		hostID, _ := strconv.ParseInt(m.HostID.ValueString(), 10, 64)
		cmd["host_id"] = hostID
		cmd["type"] = checkTypeByName[m.Type.ValueString()]
	}

	setOptionalString(cmd, "shortname", m.Shortname)
	// Only send description when non-empty — the API incorrectly uses the
	// description value to overwrite the display name (shortname). Sending
	// an empty description clears the name.
	if !m.Description.IsNull() && !m.Description.IsUnknown() && m.Description.ValueString() != "" {
		cmd["description"] = m.Description.ValueString()
	}
	setOptionalString(cmd, "address", m.Address)
	// region is stored as a name string but the API expects an integer ID.
	// Accept both named ("europe") and legacy numeric ("1") forms.
	// region=0 ("individual") is NOT sent — it is the API default and the API
	// rejects it as an invalid input value (valid inputs are 1–5 for service_add/edit).
	if !m.Region.IsNull() && !m.Region.IsUnknown() {
		val := m.Region.ValueString()
		var regionID int64
		if id, ok := regionByName[val]; ok {
			regionID = id
		} else if id, err := strconv.ParseInt(val, 10, 64); err == nil {
			regionID = id
		}
		if regionID > 0 {
			cmd["region"] = regionID
		}
	}
	setOptionalString(cmd, "message", m.Message)

	// The API returns HTTP 500 when notification flags or bool fields are sent
	// as 0. Omitting the field is equivalent to 0 (disabled/false), so we only
	// send the value when it is explicitly true.
	setOptionalBoolAsInt := func(key string, v types.Bool) {
		if !v.IsNull() && !v.IsUnknown() && v.ValueBool() {
			cmd[key] = 1
		}
	}
	setOptionalFloat := func(key string, v types.Float64) {
		if !v.IsNull() && !v.IsUnknown() {
			cmd[key] = v.ValueFloat64()
		}
	}
	setOptionalInt := func(key string, v types.Int64) {
		if !v.IsNull() && !v.IsUnknown() {
			cmd[key] = v.ValueInt64()
		}
	}

	setOptionalBoolAsInt("ipv6", m.IPv6)
	// The API rejects interval/notify_after/reminder_interval when 0; only send non-zero values.
	if !m.Interval.IsNull() && !m.Interval.IsUnknown() && m.Interval.ValueFloat64() != 0 {
		cmd["interval"] = m.Interval.ValueFloat64()
	}
	if !m.NotifyAfter.IsNull() && !m.NotifyAfter.IsUnknown() && m.NotifyAfter.ValueFloat64() != 0 {
		cmd["notify_after"] = m.NotifyAfter.ValueFloat64()
	}
	if !m.Reminder.IsNull() && !m.Reminder.IsUnknown() && m.Reminder.ValueFloat64() != 0 {
		cmd["reminder_interval"] = m.Reminder.ValueFloat64()
	}

	setOptionalBoolAsInt("email_critical", m.EmailCritical)
	setOptionalBoolAsInt("email_warning", m.EmailWarning)
	setOptionalBoolAsInt("email_ok", m.EmailOk)
	setOptionalBoolAsInt("email_remind", m.EmailRemind)
	setOptionalBoolAsInt("sms_critical", m.SmsCritical)
	setOptionalBoolAsInt("sms_warning", m.SmsWarning)
	setOptionalBoolAsInt("sms_ok", m.SmsOk)
	setOptionalBoolAsInt("sms_remind", m.SmsRemind)
	setOptionalBoolAsInt("push_critical", m.PushCritical)
	setOptionalBoolAsInt("push_warning", m.PushWarning)
	setOptionalBoolAsInt("push_ok", m.PushOk)
	setOptionalBoolAsInt("push_remind", m.PushRemind)
	setOptionalBoolAsInt("http_critical", m.HttpCritical)
	setOptionalBoolAsInt("http_warning", m.HttpWarning)
	setOptionalBoolAsInt("http_ok", m.HttpOk)
	setOptionalBoolAsInt("http_remind", m.HttpRemind)
	setOptionalBoolAsInt("phone_critical", m.PhoneCritical)
	setOptionalBoolAsInt("phone_warning", m.PhoneWarning)
	setOptionalBoolAsInt("phone_ok", m.PhoneOk)
	setOptionalBoolAsInt("phone_remind", m.PhoneRemind)

	// Type-specific parameters from the params block (sent flat at API top level)
	if p := m.Params; p != nil {
		setOptionalString(cmd, "path", p.Path)
		setOptionalInt("timeout", p.Timeout)
		setOptionalInt("port", p.Port)
		setOptionalBoolAsInt("ssl", p.Ssl)
		if !p.SslVersion.IsNull() && !p.SslVersion.IsUnknown() {
			nameToID, _ := sslVersionMaps(m.Type.ValueString())
			if id, ok := nameToID[p.SslVersion.ValueString()]; ok {
				cmd["ssl_version"] = id
			}
		}
		setOptionalBoolAsInt("check_cert", p.CheckCert)
		if !p.AuthType.IsNull() && !p.AuthType.IsUnknown() {
			if code, ok := authTypeByName[p.AuthType.ValueString()]; ok {
				cmd["auth_type"] = code
			}
		}
		setOptionalString(cmd, "login", p.Login)
		setOptionalString(cmd, "password", p.Password)
		setOptionalString(cmd, "username", p.Username)
		setOptionalString(cmd, "useragent", p.Useragent)
		setOptionalString(cmd, "host_header", p.HostHeader)
		setOptionalString(cmd, "searchitem", p.Searchitem)
		setOptionalString(cmd, "method", p.Method)
		setOptionalString(cmd, "postvars", p.Postvars)
		setOptionalString(cmd, "regex", p.Regex)
		setOptionalBoolAsInt("invert-regex", p.InvertRegex) // API uses hyphen
		setOptionalString(cmd, "header-regex", p.HeaderRegex) // API uses hyphen
		setOptionalInt("min_size", p.MinSize)
		setOptionalInt("max_size", p.MaxSize)
		setOptionalString(cmd, "addheader1", p.Addheader1)
		setOptionalString(cmd, "addheader2", p.Addheader2)
		setOptionalString(cmd, "addheader3", p.Addheader3)
		setOptionalString(cmd, "content_type", p.ContentType)
		setOptionalInt("status_code", p.StatusCode)
		setOptionalBoolAsInt("redirect", p.Redirect)
		setOptionalFloat("warning_throughput", p.WarningThroughput)
		setOptionalFloat("warning_connect", p.WarningConnect)
		setOptionalFloat("warning_roundtrip", p.WarningRoundtrip)
		setOptionalFloat("critical_throughput", p.CriticalThroughput)
		setOptionalFloat("critical_connect", p.CriticalConnect)
		setOptionalFloat("critical_roundtrip", p.CriticalRoundtrip)
		if !p.PerformanceNone.IsNull() && !p.PerformanceNone.IsUnknown() {
			if id, ok := performanceNoneByName[p.PerformanceNone.ValueString()]; ok {
				cmd["performance_none"] = id
			}
		}
		setOptionalString(cmd, "protocol", p.Protocol)
		setOptionalInt("validity_warning", p.ValidityWarning)
		setOptionalInt("validity_critical", p.ValidityCritical)
		setOptionalString(cmd, "different_ip", p.DifferentIP)
		setOptionalString(cmd, "expected_ip1", p.ExpectedIP1)
		setOptionalString(cmd, "expected_ip2", p.ExpectedIP2)
		setOptionalString(cmd, "expected_ip3", p.ExpectedIP3)
		setOptionalString(cmd, "expected_ip4", p.ExpectedIP4)
		setOptionalString(cmd, "expected_iplink", p.ExpectedIPLink)
		setOptionalString(cmd, "dns_server", p.DnsServer)
		setOptionalString(cmd, "dns_record", p.DnsRecord)
		setOptionalString(cmd, "dns_auth", p.DnsAuth)
		setOptionalInt("response_warning", p.ResponseWarning)
		setOptionalInt("response_critial", p.ResponseCritical) // API uses this spelling (typo in API)
		setOptionalString(cmd, "dns_name", p.DNSName)
		setOptionalString(cmd, "from", p.From)
		setOptionalInt("smtp_port", p.SmtpPort)
		setOptionalString(cmd, "sender", p.Sender)
		setOptionalString(cmd, "recipient", p.Recipient)
		if !p.PopType.IsNull() && !p.PopType.IsUnknown() {
			if id, ok := popTypeByName[p.PopType.ValueString()]; ok {
				cmd["pop_type"] = id
			}
		}
		setOptionalString(cmd, "pop_host", p.PopHost)
		setOptionalString(cmd, "pop_user", p.PopUser)
		setOptionalString(cmd, "pop_password", p.PopPassword)
		if !p.DeleteMode.IsNull() && !p.DeleteMode.IsUnknown() {
			if id, ok := deleteModeByName[p.DeleteMode.ValueString()]; ok {
				cmd["delete_mode"] = id
			}
		}
		setOptionalInt("time_warning", p.TimeWarning)
		setOptionalInt("time_critical", p.TimeCritical)
		setOptionalString(cmd, "smtp_user", p.SmtpUser)
		setOptionalString(cmd, "smtp_password", p.SmtpPassword)
		setOptionalBoolAsInt("smtp_tls", p.SmtpTls)
		setOptionalString(cmd, "smtp_header1", p.SmtpHeader1)
		setOptionalString(cmd, "database", p.Database)
		setOptionalString(cmd, "sql_query", p.SQLQuery)
		if !p.MongoConnection.IsNull() && !p.MongoConnection.IsUnknown() {
			if id, ok := mongoConnectionByName[p.MongoConnection.ValueString()]; ok {
				cmd["connection"] = id
			}
		}
		setOptionalString(cmd, "searchitem_ok", p.SearchItemOk)
		setOptionalString(cmd, "searchitem_warning", p.SearchItemWarning)
		setOptionalString(cmd, "searchitem_critical", p.SearchItemCritical)
		if !p.SearchItemNone.IsNull() && !p.SearchItemNone.IsUnknown() {
			if id, ok := searchItemNoneByName[p.SearchItemNone.ValueString()]; ok {
				cmd["searchitem_none"] = id
			}
		}
		setOptionalString(cmd, "basicauth_login", p.BasicauthLogin)
		setOptionalString(cmd, "basicauth_password", p.BasicauthPassword)
		setOptionalString(cmd, "debug_addon", p.DebugAddon)
		setOptionalString(cmd, "cmd", p.Command)
		setOptionalString(cmd, "token", p.Token)
		setOptionalInt("silent", p.Silent)
		setOptionalString(cmd, "cron", p.Cron)
		setOptionalString(cmd, "timezone", p.Timezone)
		setOptionalString(cmd, "def", p.Def)
		setOptionalBoolAsInt("performance", p.Performance)
		setOptionalString(cmd, "performance_name", p.PerformanceName)
		setOptionalString(cmd, "performance_regex", p.PerformanceRegex)
		setOptionalString(cmd, "performance_unit", p.PerformanceUnit)
		setOptionalInt("performance_digits", p.PerformanceDigits)
		setOptionalString(cmd, "transaction_id", p.TransactionID)
		setOptionalString(cmd, "definition", p.Definition)
		setOptionalString(cmd, "language", p.Language)
		setOptionalString(cmd, "suite_filter", p.SuiteFilter)
		setOptionalString(cmd, "port_range", p.PortRange)
		if !p.PortscanNone.IsNull() && !p.PortscanNone.IsUnknown() {
			if id, ok := portscanNoneByName[p.PortscanNone.ValueString()]; ok {
				cmd["portscan_none"] = id
			}
		}
		setOptionalString(cmd, "open_ports", p.OpenPorts)
	}

	return cmd
}

// applyCheckData updates state fields from the API response.
// When the API returns a value for a key, it always overwrites the current state
// (important for import where all fields start as null). When a key is absent
// from the API response, the current state value is preserved.
func applyCheckData(state *CheckResourceModel, raw map[string]interface{}) {
	// These helpers always write API values to state. This ensures import works
	// correctly (all fields start null and get populated from the API).
	getString := func(key string, cur types.String) types.String {
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
	getBool := func(key string, cur types.Bool) types.Bool {
		switch v := raw[key].(type) {
		case float64:
			return types.BoolValue(v == 1)
		case bool:
			return types.BoolValue(v)
		}
		return cur
	}
	getFloat := func(key string, cur types.Float64) types.Float64 {
		if v, ok := raw[key].(float64); ok {
			return types.Float64Value(v)
		}
		return cur
	}
	getInt := func(key string, cur types.Int64) types.Int64 {
		if v, ok := raw[key].(float64); ok {
			return types.Int64Value(int64(v))
		}
		return cur
	}

	// service_show returns the check type as a display name (e.g. "Intranet"),
	// not as the integer ID. Map it back to the internal name.
	if v, ok := raw["name"].(string); ok {
		if typeName, found := checkTypeByDisplayName[strings.ToLower(v)]; found {
			state.Type = types.StringValue(typeName)
		}
	}

	// "active" is always present in service_show.
	if v, ok := raw["active"].(float64); ok {
		state.Active = types.BoolValue(v == 1)
	}
	// service_show uses "notifications" (not "notification_active").
	if v, ok := raw["notifications"].(float64); ok {
		state.Notification = types.BoolValue(v == 1)
	}

	state.HostID = getString("host_id", state.HostID)
	state.Shortname = getString("shortname", state.Shortname)
	state.Description = getString("description", state.Description)
	state.Address = getString("address", state.Address)
	// region: API returns a numeric ID; normalize to the canonical name form.
	// When the API returns 0 ("individual" / no specific region), do NOT update state —
	// keep whatever the user configured (null if not set). This prevents region=0 from
	// being injected into state and later sent back to the API, which rejects it (API
	// only accepts region IDs 1–5 as valid inputs in service_add/edit).
	if v, ok := raw["region"].(float64); ok && int64(v) != 0 {
		if name, ok := regionByID[int64(v)]; ok {
			state.Region = types.StringValue(name)
		} else {
			state.Region = types.StringValue(fmt.Sprintf("%g", v))
		}
	}
	state.Message = getString("message", state.Message)
	state.IPv6 = getBool("ipv6", state.IPv6)
	// service_show uses "check_interval" instead of "interval", and returns it in
	// seconds while service_add/service_edit expect minutes.
	if v, ok := raw["check_interval"].(float64); ok {
		state.Interval = types.Float64Value(v / 60)
	}
	state.NotifyAfter = getFloat("notify_after", state.NotifyAfter)
	// service_show returns "reminder_after" in seconds; schema uses hours.
	if v, ok := raw["reminder_after"].(float64); ok {
		state.Reminder = types.Float64Value(v / 3600)
	}
	state.EmailCritical = getBool("email_critical", state.EmailCritical)
	state.EmailWarning = getBool("email_warning", state.EmailWarning)
	state.EmailOk = getBool("email_ok", state.EmailOk)
	state.EmailRemind = getBool("email_remind", state.EmailRemind)
	state.SmsCritical = getBool("sms_critical", state.SmsCritical)
	state.SmsWarning = getBool("sms_warning", state.SmsWarning)
	state.SmsOk = getBool("sms_ok", state.SmsOk)
	state.SmsRemind = getBool("sms_remind", state.SmsRemind)
	state.PushCritical = getBool("push_critical", state.PushCritical)
	state.PushWarning = getBool("push_warning", state.PushWarning)
	state.PushOk = getBool("push_ok", state.PushOk)
	state.PushRemind = getBool("push_remind", state.PushRemind)
	state.HttpCritical = getBool("http_critical", state.HttpCritical)
	state.HttpWarning = getBool("http_warning", state.HttpWarning)
	state.HttpOk = getBool("http_ok", state.HttpOk)
	state.HttpRemind = getBool("http_remind", state.HttpRemind)
	state.PhoneCritical = getBool("phone_critical", state.PhoneCritical)
	state.PhoneWarning = getBool("phone_warning", state.PhoneWarning)
	state.PhoneOk = getBool("phone_ok", state.PhoneOk)
	state.PhoneRemind = getBool("phone_remind", state.PhoneRemind)

	// Type-specific parameters into the params block
	if state.Params == nil {
		state.Params = &CheckParamsModel{}
	}
	p := state.Params
	p.Path = getString("path", p.Path)
	p.Timeout = getInt("timeout", p.Timeout)
	p.Port = getInt("port", p.Port)
	p.Ssl = getBool("ssl", p.Ssl)
	if v, ok := raw["ssl_version"].(float64); ok {
		_, idToName := sslVersionMaps(state.Type.ValueString())
		if name, found := idToName[int64(v)]; found {
			p.SslVersion = types.StringValue(name)
		} else {
			p.SslVersion = types.StringValue(fmt.Sprintf("%d", int64(v)))
		}
	}
	p.CheckCert = getBool("check_cert", p.CheckCert)
	if v, ok := raw["auth_type"].(string); ok {
		if name, found := authTypeByCode[v]; found {
			p.AuthType = types.StringValue(name)
		} else {
			p.AuthType = types.StringValue(v)
		}
	}
	p.Login = getString("login", p.Login)
	p.Password = getString("password", p.Password)
	p.Username = getString("username", p.Username)
	p.Useragent = getString("useragent", p.Useragent)
	p.HostHeader = getString("host_header", p.HostHeader)
	p.Searchitem = getString("searchitem", p.Searchitem)
	p.Method = getString("method", p.Method)
	p.Postvars = getString("postvars", p.Postvars)
	p.Regex = getString("regex", p.Regex)
	p.InvertRegex = getBool("invert-regex", p.InvertRegex)
	p.HeaderRegex = getString("header-regex", p.HeaderRegex)
	p.MinSize = getInt("min_size", p.MinSize)
	p.MaxSize = getInt("max_size", p.MaxSize)
	p.Addheader1 = getString("addheader1", p.Addheader1)
	p.Addheader2 = getString("addheader2", p.Addheader2)
	p.Addheader3 = getString("addheader3", p.Addheader3)
	p.ContentType = getString("content_type", p.ContentType)
	p.StatusCode = getInt("status_code", p.StatusCode)
	p.Redirect = getBool("redirect", p.Redirect)
	p.WarningThroughput = getFloat("warning_throughput", p.WarningThroughput)
	p.WarningConnect = getFloat("warning_connect", p.WarningConnect)
	p.WarningRoundtrip = getFloat("warning_roundtrip", p.WarningRoundtrip)
	p.CriticalThroughput = getFloat("critical_throughput", p.CriticalThroughput)
	p.CriticalConnect = getFloat("critical_connect", p.CriticalConnect)
	p.CriticalRoundtrip = getFloat("critical_roundtrip", p.CriticalRoundtrip)
	if v, ok := raw["performance_none"].(float64); ok {
		if name, found := performanceNoneByID[int64(v)]; found {
			p.PerformanceNone = types.StringValue(name)
		} else {
			p.PerformanceNone = types.StringValue(fmt.Sprintf("%d", int64(v)))
		}
	}
	p.Protocol = getString("protocol", p.Protocol)
	p.ValidityWarning = getInt("validity_warning", p.ValidityWarning)
	p.ValidityCritical = getInt("validity_critical", p.ValidityCritical)
	p.DifferentIP = getString("different_ip", p.DifferentIP)
	p.ExpectedIP1 = getString("expected_ip1", p.ExpectedIP1)
	p.ExpectedIP2 = getString("expected_ip2", p.ExpectedIP2)
	p.ExpectedIP3 = getString("expected_ip3", p.ExpectedIP3)
	p.ExpectedIP4 = getString("expected_ip4", p.ExpectedIP4)
	p.ExpectedIPLink = getString("expected_iplink", p.ExpectedIPLink)
	p.DnsServer = getString("dns_server", p.DnsServer)
	p.DnsRecord = getString("dns_record", p.DnsRecord)
	p.DnsAuth = getString("dns_auth", p.DnsAuth)
	p.ResponseWarning = getInt("response_warning", p.ResponseWarning)
	p.ResponseCritical = getInt("response_critial", p.ResponseCritical) // API uses this spelling (typo in API)
	p.DNSName = getString("dns_name", p.DNSName)
	p.From = getString("from", p.From)
	p.SmtpPort = getInt("smtp_port", p.SmtpPort)
	p.Sender = getString("sender", p.Sender)
	p.Recipient = getString("recipient", p.Recipient)
	if v, ok := raw["pop_type"].(float64); ok {
		if name, found := popTypeByID[int64(v)]; found {
			p.PopType = types.StringValue(name)
		} else {
			p.PopType = types.StringValue(fmt.Sprintf("%d", int64(v)))
		}
	}
	p.PopHost = getString("pop_host", p.PopHost)
	p.PopUser = getString("pop_user", p.PopUser)
	p.PopPassword = getString("pop_password", p.PopPassword)
	if v, ok := raw["delete_mode"].(float64); ok {
		if name, found := deleteModeByID[int64(v)]; found {
			p.DeleteMode = types.StringValue(name)
		} else {
			p.DeleteMode = types.StringValue(fmt.Sprintf("%d", int64(v)))
		}
	}
	p.TimeWarning = getInt("time_warning", p.TimeWarning)
	p.TimeCritical = getInt("time_critical", p.TimeCritical)
	p.SmtpUser = getString("smtp_user", p.SmtpUser)
	p.SmtpPassword = getString("smtp_password", p.SmtpPassword)
	p.SmtpTls = getBool("smtp_tls", p.SmtpTls)
	p.SmtpHeader1 = getString("smtp_header1", p.SmtpHeader1)
	p.Database = getString("database", p.Database)
	p.SQLQuery = getString("sql_query", p.SQLQuery)
	if v, ok := raw["connection"].(float64); ok {
		if name, found := mongoConnectionByID[int64(v)]; found {
			p.MongoConnection = types.StringValue(name)
		} else {
			p.MongoConnection = types.StringValue(fmt.Sprintf("%d", int64(v)))
		}
	}
	p.SearchItemOk = getString("searchitem_ok", p.SearchItemOk)
	p.SearchItemWarning = getString("searchitem_warning", p.SearchItemWarning)
	p.SearchItemCritical = getString("searchitem_critical", p.SearchItemCritical)
	if v, ok := raw["searchitem_none"].(float64); ok {
		if name, found := searchItemNoneByID[int64(v)]; found {
			p.SearchItemNone = types.StringValue(name)
		} else {
			p.SearchItemNone = types.StringValue(fmt.Sprintf("%d", int64(v)))
		}
	}
	p.BasicauthLogin = getString("basicauth_login", p.BasicauthLogin)
	p.BasicauthPassword = getString("basicauth_password", p.BasicauthPassword)
	p.DebugAddon = getString("debug_addon", p.DebugAddon)
	p.Command = getString("cmd", p.Command)
	p.Token = getString("token", p.Token)
	p.Silent = getInt("silent", p.Silent)
	p.Cron = getString("cron", p.Cron)
	p.Timezone = getString("timezone", p.Timezone)
	p.Def = getString("def", p.Def)
	p.Performance = getBool("performance", p.Performance)
	p.PerformanceName = getString("performance_name", p.PerformanceName)
	p.PerformanceRegex = getString("performance_regex", p.PerformanceRegex)
	p.PerformanceUnit = getString("performance_unit", p.PerformanceUnit)
	p.PerformanceDigits = getInt("performance_digits", p.PerformanceDigits)
	p.TransactionID = getString("transaction_id", p.TransactionID)
	p.Definition = getString("definition", p.Definition)
	p.Language = getString("language", p.Language)
	p.SuiteFilter = getString("suite_filter", p.SuiteFilter)
	p.PortRange = getString("port_range", p.PortRange)
	if v, ok := raw["portscan_none"].(float64); ok {
		if name, found := portscanNoneByID[int64(v)]; found {
			p.PortscanNone = types.StringValue(name)
		} else {
			p.PortscanNone = types.StringValue(fmt.Sprintf("%d", int64(v)))
		}
	}
	p.OpenPorts = getString("open_ports", p.OpenPorts)

	// If all params are null (e.g. blacklist type), set params to nil
	// to avoid an empty params block in state.
	allNull := true
	for _, pc := range paramsSetFields(p) {
		if pc.isSet {
			allNull = false
			break
		}
	}
	if allNull {
		state.Params = nil
	}
}

// applyMonitoring calls service_monitoring to enable or disable check execution.
// When monitoring is null/unknown (not configured), it defaults to enabled (1).
func (r *CheckResource) applyMonitoring(ctx context.Context, id string, monitoring types.Bool) error {
	val := 1
	if !monitoring.IsNull() && !monitoring.IsUnknown() && !monitoring.ValueBool() {
		val = 0
	}
	_, err := r.client.Do(ctx, client.Request{
		"command":    "service_monitoring",
		"id":         id,
		"monitoring": val,
	})
	return err
}

// applyNotification calls service_notification to enable or disable alert notifications.
// When notification is null/unknown (not configured), it defaults to enabled (1).
func (r *CheckResource) applyNotification(ctx context.Context, id string, notification types.Bool) error {
	val := 1
	if !notification.IsNull() && !notification.IsUnknown() && !notification.ValueBool() {
		val = 0
	}
	_, err := r.client.Do(ctx, client.Request{
		"command":      "service_notification",
		"id":           id,
		"notification": val,
	})
	return err
}
