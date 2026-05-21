package provider

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Snapshot tests for buildCheckCommand and applyCheckData. The point is to
// freeze the current encode/decode behaviour as a golden file so a refactor
// (e.g. moving to a table-driven field manifest) can be verified to produce
// byte-identical output.
//
// Regenerate the golden files with:
//
//	go test ./internal/provider/ -run Snapshot -update-golden

var updateGolden = flag.Bool("update-golden", false, "rewrite golden files instead of comparing")

// kitchenSinkModel returns a CheckResourceModel with every field populated
// with a deterministic non-default value, so the snapshot covers every encode
// path including the per-type enum mappings.
//
// Type is set to "http" so ssl_version uses the HTTP map (auto=1).
func kitchenSinkModel() CheckResourceModel {
	return CheckResourceModel{
		ID:           types.StringValue("123"),
		HostID:       types.StringValue("456"),
		Type:         types.StringValue("http"),
		Shortname:    types.StringValue("my-check"),
		Description:  types.StringValue("description text"),
		Address:      types.StringValue("example.com"),
		Region:       types.StringValue("europe"),
		IPv6:         types.BoolValue(true),
		Interval:     types.Float64Value(5),
		NotifyAfter:  types.Float64Value(10),
		Reminder:     types.Float64Value(24),
		Message:      types.StringValue("alert message"),
		Active:       types.BoolValue(true),
		Notification: types.BoolValue(true),

		EmailCritical: types.BoolValue(true),
		EmailWarning:  types.BoolValue(true),
		EmailOk:       types.BoolValue(true),
		EmailRemind:   types.BoolValue(true),
		SmsCritical:   types.BoolValue(true),
		SmsWarning:    types.BoolValue(true),
		SmsOk:         types.BoolValue(true),
		SmsRemind:     types.BoolValue(true),
		PushCritical:  types.BoolValue(true),
		PushWarning:   types.BoolValue(true),
		PushOk:        types.BoolValue(true),
		PushRemind:    types.BoolValue(true),
		HttpCritical:  types.BoolValue(true),
		HttpWarning:   types.BoolValue(true),
		HttpOk:        types.BoolValue(true),
		HttpRemind:    types.BoolValue(true),
		PhoneCritical: types.BoolValue(true),
		PhoneWarning:  types.BoolValue(true),
		PhoneOk:       types.BoolValue(true),
		PhoneRemind:   types.BoolValue(true),

		Params: &CheckParamsModel{
			Path:               types.StringValue("/health"),
			Timeout:            types.Int64Value(30),
			Port:               types.Int64Value(443),
			Ssl:                types.BoolValue(true),
			SslVersion:         types.StringValue("auto"),
			CheckCert:          types.BoolValue(true),
			AuthType:           types.StringValue("basic"),
			Login:              types.StringValue("alice"),
			Password:           types.StringValue("s3cret"),
			Username:           types.StringValue("bob"),
			Useragent:          types.StringValue("ua/1.0"),
			HostHeader:         types.StringValue("vhost.example.com"),
			Searchitem:         types.StringValue("needle"),
			Method:             types.StringValue("POST"),
			Postvars:           types.StringValue("a=1&b=2"),
			Regex:              types.StringValue(".*"),
			InvertRegex:        types.BoolValue(true),
			HeaderRegex:        types.StringValue("^X-"),
			MinSize:            types.Int64Value(100),
			MaxSize:            types.Int64Value(1000),
			Addheader1:         types.StringValue("X-1: a"),
			Addheader2:         types.StringValue("X-2: b"),
			Addheader3:         types.StringValue("X-3: c"),
			ContentType:        types.StringValue("application/json"),
			StatusCode:         types.Int64Value(200),
			Redirect:           types.BoolValue(true),
			WarningThroughput:  types.Float64Value(1.5),
			WarningConnect:     types.Float64Value(2.5),
			WarningRoundtrip:   types.Float64Value(3.5),
			CriticalThroughput: types.Float64Value(4.5),
			CriticalConnect:    types.Float64Value(5.5),
			CriticalRoundtrip:  types.Float64Value(6.5),
			PerformanceNone:    types.StringValue("critical"),
			Protocol:           types.StringValue("https"),
			ValidityWarning:    types.Int64Value(14),
			ValidityCritical:   types.Int64Value(7),
			DifferentIP:        types.StringValue("0"),
			ExpectedIP1:        types.StringValue("1.2.3.4"),
			ExpectedIP2:        types.StringValue("5.6.7.8"),
			ExpectedIP3:        types.StringValue("9.10.11.12"),
			ExpectedIP4:        types.StringValue("13.14.15.16"),
			ExpectedIPLink:     types.StringValue("http://link"),
			DnsServer:          types.StringValue("8.8.8.8"),
			DnsRecord:          types.StringValue("A"),
			DnsAuth:            types.StringValue("dnskey"),
			ResponseWarning:    types.Int64Value(100),
			ResponseCritical:   types.Int64Value(500),
			DNSName:            types.StringValue("example.com"),
			From:               types.StringValue("from@example.com"),
			SmtpPort:           types.Int64Value(587),
			Sender:             types.StringValue("send@example.com"),
			Recipient:          types.StringValue("rcv@example.com"),
			PopType:            types.StringValue("pop3s"),
			PopHost:            types.StringValue("pop.example.com"),
			PopUser:            types.StringValue("popuser"),
			PopPassword:        types.StringValue("poppw"),
			DeleteMode:         types.StringValue("all"),
			TimeWarning:        types.Int64Value(15),
			TimeCritical:       types.Int64Value(60),
			SmtpUser:           types.StringValue("smtpuser"),
			SmtpPassword:       types.StringValue("smtppw"),
			SmtpTls:            types.BoolValue(true),
			SmtpHeader1:        types.StringValue("X-Header: v"),
			Database:           types.StringValue("appdb"),
			SQLQuery:           types.StringValue("SELECT 1"),
			MongoConnection:    types.StringValue("mongodb_srv"),
			SearchItemOk:       types.StringValue("ok"),
			SearchItemWarning:  types.StringValue("warn"),
			SearchItemCritical: types.StringValue("crit"),
			SearchItemNone:     types.StringValue("warning"),
			BasicauthLogin:     types.StringValue("baLogin"),
			BasicauthPassword:  types.StringValue("baPass"),
			DebugAddon:         types.StringValue("debug=1"),
			Command:            types.StringValue("uptime"),
			Token:              types.StringValue("tok123"),
			Silent:             types.Int64Value(1),
			Cron:               types.StringValue("*/5 * * * *"),
			Timezone:           types.StringValue("Europe/Berlin"),
			Def:                types.StringValue("def-value"),
			Performance:        types.BoolValue(true),
			PerformanceName:    types.StringValue("perfName"),
			PerformanceRegex:   types.StringValue("perfRegex"),
			PerformanceUnit:    types.StringValue("ms"),
			PerformanceDigits:  types.Int64Value(2),
			TransactionID:      types.StringValue("tx-1"),
			Definition:         types.StringValue("def-script"),
			Language:           types.StringValue("en"),
			SuiteFilter:        types.StringValue("smoke"),
			PortRange:          types.StringValue("1-1024"),
			PortscanNone:       types.StringValue("critical"),
			OpenPorts:          types.StringValue("80,443"),
		},
	}
}

// kitchenSinkRaw mirrors the flattened service_show response that
// refreshFromAPI hands to applyCheckData. Keys, value types, and special
// transformations (interval in seconds, reminder_after in seconds, "name" as
// the human display string, "cmd" / "connection" / "response_critial" / etc.)
// match what the Serverguard24 API actually emits.
func kitchenSinkRaw() map[string]any {
	return map[string]any{
		"name":           "http",
		"active":         float64(1),
		"notifications":  float64(1),
		"host_id":        "456",
		"shortname":      "my-check",
		"description":    "description text",
		"address":        "example.com",
		"region":         float64(1), // europe
		"message":        "alert message",
		"ipv6":           float64(1),
		"check_interval": float64(300), // 5 minutes in seconds
		"notify_after":   float64(10),
		"reminder_after": float64(86400), // 24 hours in seconds

		"email_critical": float64(1),
		"email_warning":  float64(1),
		"email_ok":       float64(1),
		"email_remind":   float64(1),
		"sms_critical":   float64(1),
		"sms_warning":    float64(1),
		"sms_ok":         float64(1),
		"sms_remind":     float64(1),
		"push_critical":  float64(1),
		"push_warning":   float64(1),
		"push_ok":        float64(1),
		"push_remind":    float64(1),
		"http_critical":  float64(1),
		"http_warning":   float64(1),
		"http_ok":        float64(1),
		"http_remind":    float64(1),
		"phone_critical": float64(1),
		"phone_warning":  float64(1),
		"phone_ok":       float64(1),
		"phone_remind":   float64(1),

		"path":                "/health",
		"timeout":             float64(30),
		"port":                float64(443),
		"ssl":                 float64(1),
		"ssl_version":         float64(1), // auto in HTTP map
		"check_cert":          float64(1),
		"auth_type":           "b", // basic
		"login":               "alice",
		"password":            "s3cret",
		"username":            "bob",
		"useragent":           "ua/1.0",
		"host_header":         "vhost.example.com",
		"searchitem":          "needle",
		"method":              "POST",
		"postvars":            "a=1&b=2",
		"regex":               ".*",
		"invert-regex":        float64(1),
		"header-regex":        "^X-",
		"min_size":            float64(100),
		"max_size":            float64(1000),
		"addheader1":          "X-1: a",
		"addheader2":          "X-2: b",
		"addheader3":          "X-3: c",
		"content_type":        "application/json",
		"status_code":         float64(200),
		"redirect":            float64(1),
		"warning_throughput":  1.5,
		"warning_connect":     2.5,
		"warning_roundtrip":   3.5,
		"critical_throughput": 4.5,
		"critical_connect":    5.5,
		"critical_roundtrip":  6.5,
		"performance_none":    float64(1), // critical
		"protocol":            "https",
		"validity_warning":    float64(14),
		"validity_critical":   float64(7),
		"different_ip":        "0",
		"expected_ip1":        "1.2.3.4",
		"expected_ip2":        "5.6.7.8",
		"expected_ip3":        "9.10.11.12",
		"expected_ip4":        "13.14.15.16",
		"expected_iplink":     "http://link",
		"dns_server":          "8.8.8.8",
		"dns_record":          "A",
		"dns_auth":            "dnskey",
		"response_warning":    float64(100),
		"response_critial":    float64(500), // API typo
		"dns_name":            "example.com",
		"from":                "from@example.com",
		"smtp_port":           float64(587),
		"sender":              "send@example.com",
		"recipient":           "rcv@example.com",
		"pop_type":            float64(995), // pop3s
		"pop_host":            "pop.example.com",
		"pop_user":            "popuser",
		"pop_password":        "poppw",
		"delete_mode":         float64(3), // all
		"time_warning":        float64(15),
		"time_critical":       float64(60),
		"smtp_user":           "smtpuser",
		"smtp_password":       "smtppw",
		"smtp_tls":            float64(1),
		"smtp_header1":        "X-Header: v",
		"database":            "appdb",
		"sql_query":           "SELECT 1",
		"connection":          float64(1), // mongodb_srv
		"searchitem_ok":       "ok",
		"searchitem_warning":  "warn",
		"searchitem_critical": "crit",
		"searchitem_none":     float64(1), // warning
		"basicauth_login":     "baLogin",
		"basicauth_password":  "baPass",
		"debug_addon":         "debug=1",
		"cmd":                 "uptime",
		"token":               "tok123",
		"silent":              float64(1),
		"cron":                "*/5 * * * *",
		"timezone":            "Europe/Berlin",
		"def":                 "def-value",
		"performance":         float64(1),
		"performance_name":    "perfName",
		"performance_regex":   "perfRegex",
		"performance_unit":    "ms",
		"performance_digits":  float64(2),
		"transaction_id":      "tx-1",
		"definition":          "def-script",
		"language":            "en",
		"suite_filter":        "smoke",
		"port_range":          "1-1024",
		"portscan_none":       float64(2), // critical
		"open_ports":          "80,443",
	}
}

// canonicalJSON serialises v with map keys sorted (Go's encoding/json sorts
// map[string]X by key) and struct fields in declaration order.
func canonicalJSON(t *testing.T, v any) []byte {
	t.Helper()
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
	}
	return append(out, '\n')
}

// modelAsMap reflects over CheckResourceModel (and nested CheckParamsModel)
// reducing types.X wrappers to plain Go values so the result can be
// JSON-encoded for stable golden comparison. types.X values that are null or
// unknown become nil.
func modelAsMap(v any) map[string]any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	out := make(map[string]any, rv.NumField())
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		ft := rt.Field(i)
		tag := ft.Tag.Get("tfsdk")
		if tag == "" || tag == "-" {
			continue
		}
		out[tag] = scalarToJSON(rv.Field(i).Interface())
	}
	return out
}

func scalarToJSON(v any) any {
	switch x := v.(type) {
	case types.String:
		if x.IsNull() || x.IsUnknown() {
			return nil
		}
		return x.ValueString()
	case types.Bool:
		if x.IsNull() || x.IsUnknown() {
			return nil
		}
		return x.ValueBool()
	case types.Int64:
		if x.IsNull() || x.IsUnknown() {
			return nil
		}
		return x.ValueInt64()
	case types.Float64:
		if x.IsNull() || x.IsUnknown() {
			return nil
		}
		return x.ValueFloat64()
	case *CheckParamsModel:
		if x == nil {
			return nil
		}
		return modelAsMap(x)
	}
	return v
}

// assertGolden compares got against testdata/<name>. When -update-golden is
// set, writes got into the file instead.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update-golden to create)", path, err)
	}
	if string(got) != string(want) {
		t.Errorf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}

func TestBuildCheckCommand_KitchenSink_ServiceAdd(t *testing.T) {
	cmd := buildCheckCommand("service_add", "", kitchenSinkModel())
	assertGolden(t, "buildCheckCommand_service_add.golden.json", canonicalJSON(t, cmd))
}

func TestBuildCheckCommand_KitchenSink_ServiceEdit(t *testing.T) {
	cmd := buildCheckCommand("service_edit", "789", kitchenSinkModel())
	assertGolden(t, "buildCheckCommand_service_edit.golden.json", canonicalJSON(t, cmd))
}

func TestApplyCheckData_KitchenSink(t *testing.T) {
	state := CheckResourceModel{}
	applyCheckData(&state, kitchenSinkRaw())
	assertGolden(t, "applyCheckData_kitchen_sink.golden.json", canonicalJSON(t, modelAsMap(&state)))
}

func TestBuildCheckCommand_SSLVersionPerType(t *testing.T) {
	cases := []struct {
		checkType string
		sslVer    string
		wantID    int64
	}{
		{"http", "auto", 1},
		{"http", "tlsv1", 2},
		{"http", "sslv3", 3},
		{"http_perf", "auto", 1},
		{"ssl_cert", "auto", 0},
		{"ssl_cert", "tlsv1", 1},
		{"ssl_cert", "sslv3", 2},
		{"smtp", "none", 0},
		{"smtp", "tls", 1},
		{"smtp", "ssl", 2},
	}
	for _, tc := range cases {
		t.Run(tc.checkType+"_"+tc.sslVer, func(t *testing.T) {
			m := CheckResourceModel{
				HostID: types.StringValue("1"),
				Type:   types.StringValue(tc.checkType),
				Params: &CheckParamsModel{SslVersion: types.StringValue(tc.sslVer)},
			}
			cmd := buildCheckCommand("service_add", "", m)
			got, ok := cmd["ssl_version"].(int64)
			if !ok {
				t.Fatalf("ssl_version missing or wrong type: %T", cmd["ssl_version"])
			}
			if got != tc.wantID {
				t.Errorf("ssl_version = %d, want %d", got, tc.wantID)
			}
		})
	}
}

func TestApplyCheckData_SSLVersionPerType(t *testing.T) {
	cases := []struct {
		checkType string
		rawID     float64
		wantName  string
	}{
		{"http", 1, "auto"},
		{"http", 2, "tlsv1"},
		{"http", 3, "sslv3"},
		{"http_perf", 1, "auto"},
		{"ssl_cert", 0, "auto"},
		{"ssl_cert", 1, "tlsv1"},
		{"ssl_cert", 2, "sslv3"},
		{"smtp", 0, "none"},
		{"smtp", 1, "tls"},
		{"smtp", 2, "ssl"},
	}
	for _, tc := range cases {
		t.Run(tc.checkType+"_"+tc.wantName, func(t *testing.T) {
			state := CheckResourceModel{Type: types.StringValue(tc.checkType)}
			applyCheckData(&state, map[string]any{"ssl_version": tc.rawID})
			if state.Params == nil || state.Params.SslVersion.ValueString() != tc.wantName {
				t.Errorf("ssl_version = %q, want %q", state.Params.SslVersion.ValueString(), tc.wantName)
			}
		})
	}
}
