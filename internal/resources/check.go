package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

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

type CheckResource struct {
	client *client.Client
}

// CheckResourceModel holds all possible attributes for all check types.
// Type-specific fields are set only when relevant.
type CheckResourceModel struct {
	ID          types.String  `tfsdk:"id"`
	HostID      types.String `tfsdk:"host_id"`
	Type        types.String `tfsdk:"type"`
	Shortname   types.String  `tfsdk:"shortname"`
	Description types.String  `tfsdk:"description"`
	Address     types.String  `tfsdk:"address"`
	Region      types.String  `tfsdk:"region"`
	IPv6        types.Bool    `tfsdk:"ipv6"`
	Interval    types.Float64 `tfsdk:"interval"`
	NotifyAfter types.Float64 `tfsdk:"notify_after"`
	Reminder    types.Float64 `tfsdk:"reminder_interval"`
	Message     types.String  `tfsdk:"message"`
	Active       types.Bool `tfsdk:"active"`
	Notification types.Bool `tfsdk:"notification"`

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

	// Type-specific fields
	Path                types.String  `tfsdk:"path"`
	Timeout             types.Int64   `tfsdk:"timeout"`
	Port                types.Int64   `tfsdk:"port"`
	WarningThroughput   types.Float64 `tfsdk:"warning_throughput"`
	WarningConnect      types.Float64 `tfsdk:"warning_connect"`
	WarningRoundtrip    types.Float64 `tfsdk:"warning_roundtrip"`
	CriticalThroughput  types.Float64 `tfsdk:"critical_throughput"`
	CriticalConnect     types.Float64 `tfsdk:"critical_connect"`
	CriticalRoundtrip   types.Float64 `tfsdk:"critical_roundtrip"`
	Protocol            types.String  `tfsdk:"protocol"`
	ValidityWarning     types.Int64   `tfsdk:"validity_warning"`
	ValidityCritical    types.Int64   `tfsdk:"validity_critical"`
	ExpectedIP1         types.String  `tfsdk:"expected_ip1"`
	ResponseWarning     types.Int64   `tfsdk:"response_warning"`
	ResponseCritical    types.Int64   `tfsdk:"response_critical"`
	DNSName             types.String  `tfsdk:"dns_name"`
	SmtpPort            types.Int64   `tfsdk:"smtp_port"`
	Sender              types.String  `tfsdk:"sender"`
	Recipient           types.String  `tfsdk:"recipient"`
	PopType             types.String  `tfsdk:"pop_type"`
	PopHost             types.String  `tfsdk:"pop_host"`
	PopUser             types.String  `tfsdk:"pop_user"`
	PopPassword         types.String  `tfsdk:"pop_password"`
	DeleteMode          types.String  `tfsdk:"delete_mode"`
	TimeWarning         types.Int64   `tfsdk:"time_warning"`
	TimeCritical        types.Int64   `tfsdk:"time_critical"`
	Username            types.String  `tfsdk:"username"`
	Password            types.String  `tfsdk:"password"`
	Database            types.String  `tfsdk:"database"`
	SQLQuery            types.String  `tfsdk:"sql_query"`
	MongoConnection     types.String  `tfsdk:"mongo_connection"`
	SearchItemOk        types.String  `tfsdk:"searchitem_ok"`
	SearchItemCritical  types.String  `tfsdk:"searchitem_critical"`
	SearchItemNone      types.Bool    `tfsdk:"searchitem_none"`
	Command             types.String  `tfsdk:"command"`
	Cron                types.String  `tfsdk:"cron"`
	Timezone            types.String  `tfsdk:"timezone"`
	Token               types.String  `tfsdk:"token"`
	TransactionID       types.String  `tfsdk:"transaction_id"`
	Definition          types.String  `tfsdk:"definition"`
	Language            types.String  `tfsdk:"language"`
	PortRange           types.String  `tfsdk:"port_range"`
	PortscanNone        types.Bool    `tfsdk:"portscan_none"`
}

func NewCheckResource() resource.Resource {
	return &CheckResource{}
}

func (r *CheckResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check"
}

func (r *CheckResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	boolOpt := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Description: desc, MarkdownDescription: desc}
	}
	strOpt := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Description: desc, MarkdownDescription: desc}
	}
	intOpt := func(desc string) schema.Int64Attribute {
		return schema.Int64Attribute{Optional: true, Description: desc, MarkdownDescription: desc}
	}
	floatOpt := func(desc string) schema.Float64Attribute {
		return schema.Float64Attribute{Optional: true, Description: desc, MarkdownDescription: desc}
	}

	resp.Schema = schema.Schema{
		Description:         "Manages a Serverguard24 monitoring check.",
		MarkdownDescription: "Manages a Serverguard24 monitoring check.",
		Version:             0,
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
			"shortname":        strOpt("Short display name (max 40 characters)."),
			"description":      strOpt("Description of this check."),
			"address":          strOpt("Override address for this check."),
			"region":           strOpt("Monitoring region: 1=Europe, 2=Americas, 3=Europe+Americas, 4=Asia/Pacific, 5=World."),
			"ipv6":             boolOpt("Use IPv6."),
			"interval":         floatOpt("Check interval in minutes."),
			"notify_after":     floatOpt("Notify after N minutes of downtime."),
			"reminder_interval": floatOpt("Reminder interval in hours."),
			"message":          strOpt("Custom notification message."),
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				Description:         "Set to false to deactivate the check without deleting it. Defaults to true.",
				MarkdownDescription: "Set to `false` to deactivate the check without deleting it. Defaults to `true`.",
			},
			"notification": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
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
			// Type-specific
			"path":                 strOpt("URL path (HTTP checks)."),
			"timeout":              intOpt("Timeout in seconds."),
			"port":                 intOpt("Port number."),
			"warning_throughput":   floatOpt("Warning threshold for throughput (KB/s)."),
			"warning_connect":      floatOpt("Warning threshold for connect time (ms)."),
			"warning_roundtrip":    floatOpt("Warning threshold for roundtrip time (ms)."),
			"critical_throughput":  floatOpt("Critical threshold for throughput (KB/s)."),
			"critical_connect":     floatOpt("Critical threshold for connect time (ms)."),
			"critical_roundtrip":   floatOpt("Critical threshold for roundtrip time (ms)."),
			"protocol":             strOpt("Protocol (SSL Cert check)."),
			"validity_warning":     intOpt("Warn when certificate expires within N days."),
			"validity_critical":    intOpt("Alert when certificate expires within N days."),
			"expected_ip1":         strOpt("Expected IP for domain check."),
			"response_warning":     intOpt("Warning response time (ms) for PING."),
			"response_critical":    intOpt("Critical response time (ms) for PING."),
			"dns_name":             strOpt("DNS name to resolve."),
			"smtp_port":            intOpt("SMTP port for email checks."),
			"sender":               strOpt("Sender address for email checks."),
			"recipient":            strOpt("Recipient address for email checks."),
			"pop_type":             strOpt("POP type for email checks."),
			"pop_host":             strOpt("POP host for email checks."),
			"pop_user":             strOpt("POP username for email checks."),
			"pop_password":         schema.StringAttribute{Optional: true, Sensitive: true, Description: "POP password for email checks.", MarkdownDescription: "POP password for email checks."},
			"delete_mode":          strOpt("Delete mode for email checks."),
			"time_warning":         intOpt("Warning delivery time (s) for email checks."),
			"time_critical":        intOpt("Critical delivery time (s) for email checks."),
			"username":             strOpt("Username for database checks."),
			"password":             schema.StringAttribute{Optional: true, Sensitive: true, Description: "Password for database checks.", MarkdownDescription: "Password for database checks."},
			"database":             strOpt("Database name."),
			"sql_query":            strOpt("SQL query to execute."),
			"mongo_connection":     strOpt("MongoDB connection string."),
			"searchitem_ok":        strOpt("String to search for in OK state."),
			"searchitem_critical":  strOpt("String to search for in critical state."),
			"searchitem_none":      boolOpt("Alert if search item is not found."),
			"command":              strOpt("Command to run (local/intranet checks)."),
			"cron":                 strOpt("Cron expression for cron checks."),
			"timezone":             strOpt("Timezone for cron checks."),
			"token":                schema.StringAttribute{Optional: true, Sensitive: true, Description: "Token for system checks.", MarkdownDescription: "Token for system checks."},
			"transaction_id":       strOpt("Transaction ID for TransAct checks."),
			"definition":           strOpt("Script definition (Selenium/Postman checks)."),
			"language":             strOpt("Script language (Selenium checks)."),
			"port_range":           strOpt("Port range for port scan checks."),
			"portscan_none":        boolOpt("Alert if no open ports found."),
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

func (r *CheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CheckResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := buildCheckCommand("service_add", "", plan)
	apiResp, err := r.client.Do(cmd)
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

	if err := r.applyMonitoring(plan.ID.ValueString(), plan.Active); err != nil {
		resp.Diagnostics.AddError("Error setting active state", err.Error())
		return
	}
	if err := r.applyNotification(plan.ID.ValueString(), plan.Notification); err != nil {
		resp.Diagnostics.AddError("Error setting notification state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CheckResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.Do(client.Request{
		"command": "service_show",
		"id":      state.ID.ValueString(),
	})
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading check", err.Error())
		return
	}

	// service_show returns a single object. Flatten nested sub-objects so that
	// applyCheckData can access all fields at the top level.
	var raw map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &raw); err != nil {
		resp.Diagnostics.AddError("Error parsing check data", err.Error())
		return
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

	applyCheckData(&state, raw)
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
	if _, err := r.client.Do(cmd); err != nil {
		resp.Diagnostics.AddError("Error updating check", err.Error())
		return
	}

	plan.ID = state.ID

	if err := r.applyMonitoring(plan.ID.ValueString(), plan.Active); err != nil {
		resp.Diagnostics.AddError("Error setting active state", err.Error())
		return
	}
	if err := r.applyNotification(plan.ID.ValueString(), plan.Notification); err != nil {
		resp.Diagnostics.AddError("Error setting notification state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CheckResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Do(client.Request{
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
	setOptionalString(cmd, "description", m.Description)
	setOptionalString(cmd, "address", m.Address)
	setOptionalString(cmd, "region", m.Region)
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
	setOptionalFloat("interval", m.Interval)
	setOptionalFloat("notify_after", m.NotifyAfter)
	setOptionalFloat("reminder_interval", m.Reminder)

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

	setOptionalString(cmd, "path", m.Path)
	setOptionalInt("timeout", m.Timeout)
	setOptionalInt("port", m.Port)
	setOptionalFloat("warning_throughput", m.WarningThroughput)
	setOptionalFloat("warning_connect", m.WarningConnect)
	setOptionalFloat("warning_roundtrip", m.WarningRoundtrip)
	setOptionalFloat("critical_throughput", m.CriticalThroughput)
	setOptionalFloat("critical_connect", m.CriticalConnect)
	setOptionalFloat("critical_roundtrip", m.CriticalRoundtrip)
	setOptionalString(cmd, "protocol", m.Protocol)
	setOptionalInt("validity_warning", m.ValidityWarning)
	setOptionalInt("validity_critical", m.ValidityCritical)
	setOptionalString(cmd, "expected_ip1", m.ExpectedIP1)
	setOptionalInt("response_warning", m.ResponseWarning)
	setOptionalInt("response_critial", m.ResponseCritical) // API uses this spelling (typo in API)
	setOptionalString(cmd, "dns_name", m.DNSName)
	setOptionalInt("smtp_port", m.SmtpPort)
	setOptionalString(cmd, "sender", m.Sender)
	setOptionalString(cmd, "recipient", m.Recipient)
	setOptionalString(cmd, "pop_type", m.PopType)
	setOptionalString(cmd, "pop_host", m.PopHost)
	setOptionalString(cmd, "pop_user", m.PopUser)
	setOptionalString(cmd, "pop_password", m.PopPassword)
	setOptionalString(cmd, "delete_mode", m.DeleteMode)
	setOptionalInt("time_warning", m.TimeWarning)
	setOptionalInt("time_critical", m.TimeCritical)
	setOptionalString(cmd, "username", m.Username)
	setOptionalString(cmd, "password", m.Password)
	setOptionalString(cmd, "database", m.Database)
	setOptionalString(cmd, "sql_query", m.SQLQuery)
	setOptionalString(cmd, "connection", m.MongoConnection)
	setOptionalString(cmd, "searchitem_ok", m.SearchItemOk)
	setOptionalString(cmd, "searchitem_critical", m.SearchItemCritical)
	setOptionalBoolAsInt("searchitem_none", m.SearchItemNone)
	setOptionalString(cmd, "command", m.Command)
	setOptionalString(cmd, "cron", m.Cron)
	setOptionalString(cmd, "timezone", m.Timezone)
	setOptionalString(cmd, "token", m.Token)
	setOptionalString(cmd, "transaction_id", m.TransactionID)
	setOptionalString(cmd, "definition", m.Definition)
	setOptionalString(cmd, "language", m.Language)
	setOptionalString(cmd, "port_range", m.PortRange)
	setOptionalBoolAsInt("portscan_none", m.PortscanNone)

	return cmd
}

// applyCheckData updates state fields from the API response.
// Fields that are null in the current state (not configured by the user) are
// preserved as null so Terraform does not produce spurious diffs.
func applyCheckData(state *CheckResourceModel, raw map[string]interface{}) {
	getString := func(key string, cur types.String) types.String {
		if cur.IsNull() || cur.IsUnknown() {
			return cur
		}
		if v, ok := raw[key].(string); ok {
			return types.StringValue(v)
		}
		return cur // key absent in API response — preserve configured value
	}
	getBool := func(key string, cur types.Bool) types.Bool {
		if cur.IsNull() || cur.IsUnknown() {
			return cur
		}
		switch v := raw[key].(type) {
		case float64:
			return types.BoolValue(v == 1)
		case bool:
			return types.BoolValue(v)
		}
		return cur // key absent in API response — preserve configured value
	}
	getFloat := func(key string, cur types.Float64) types.Float64 {
		if cur.IsNull() || cur.IsUnknown() {
			return cur
		}
		if v, ok := raw[key].(float64); ok {
			return types.Float64Value(v)
		}
		return cur // key absent in API response — preserve configured value
	}
	getInt := func(key string, cur types.Int64) types.Int64 {
		if cur.IsNull() || cur.IsUnknown() {
			return cur
		}
		if v, ok := raw[key].(float64); ok {
			return types.Int64Value(int64(v))
		}
		return cur // key absent in API response — preserve configured value
	}

	// "active" and "notification_active" are always present in service_show.
	if v, ok := raw["active"].(float64); ok {
		state.Active = types.BoolValue(v == 1)
	}
	if v, ok := raw["notification_active"].(float64); ok {
		state.Notification = types.BoolValue(v == 1)
	}

	state.Shortname = getString("shortname", state.Shortname)
	state.Description = getString("description", state.Description)
	state.Address = getString("address", state.Address)
	state.Region = getString("region", state.Region)
	state.Message = getString("message", state.Message)
	state.IPv6 = getBool("ipv6", state.IPv6)
	state.Interval = getFloat("interval", state.Interval)
	state.NotifyAfter = getFloat("notify_after", state.NotifyAfter)
	state.Reminder = getFloat("reminder_interval", state.Reminder)
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
	state.Path = getString("path", state.Path)
	state.Timeout = getInt("timeout", state.Timeout)
	state.Port = getInt("port", state.Port)
	state.WarningThroughput = getFloat("warning_throughput", state.WarningThroughput)
	state.WarningConnect = getFloat("warning_connect", state.WarningConnect)
	state.WarningRoundtrip = getFloat("warning_roundtrip", state.WarningRoundtrip)
	state.CriticalThroughput = getFloat("critical_throughput", state.CriticalThroughput)
	state.CriticalConnect = getFloat("critical_connect", state.CriticalConnect)
	state.CriticalRoundtrip = getFloat("critical_roundtrip", state.CriticalRoundtrip)
	state.Protocol = getString("protocol", state.Protocol)
	state.ValidityWarning = getInt("validity_warning", state.ValidityWarning)
	state.ValidityCritical = getInt("validity_critical", state.ValidityCritical)
	state.ExpectedIP1 = getString("expected_ip1", state.ExpectedIP1)
	state.ResponseWarning = getInt("response_warning", state.ResponseWarning)
	state.ResponseCritical = getInt("response_critial", state.ResponseCritical) // API uses this spelling (typo in API)
	state.DNSName = getString("dns_name", state.DNSName)
	state.SmtpPort = getInt("smtp_port", state.SmtpPort)
	state.Sender = getString("sender", state.Sender)
	state.Recipient = getString("recipient", state.Recipient)
	state.PopType = getString("pop_type", state.PopType)
	state.PopHost = getString("pop_host", state.PopHost)
	state.PopUser = getString("pop_user", state.PopUser)
	state.PopPassword = getString("pop_password", state.PopPassword)
	state.DeleteMode = getString("delete_mode", state.DeleteMode)
	state.TimeWarning = getInt("time_warning", state.TimeWarning)
	state.TimeCritical = getInt("time_critical", state.TimeCritical)
	state.Username = getString("username", state.Username)
	state.Password = getString("password", state.Password)
	state.Database = getString("database", state.Database)
	state.SQLQuery = getString("sql_query", state.SQLQuery)
	state.MongoConnection = getString("connection", state.MongoConnection)
	state.SearchItemOk = getString("searchitem_ok", state.SearchItemOk)
	state.SearchItemCritical = getString("searchitem_critical", state.SearchItemCritical)
	state.SearchItemNone = getBool("searchitem_none", state.SearchItemNone)
	state.Command = getString("command", state.Command)
	state.Cron = getString("cron", state.Cron)
	state.Timezone = getString("timezone", state.Timezone)
	state.Token = getString("token", state.Token)
	state.TransactionID = getString("transaction_id", state.TransactionID)
	state.Definition = getString("definition", state.Definition)
	state.Language = getString("language", state.Language)
	state.PortRange = getString("port_range", state.PortRange)
	state.PortscanNone = getBool("portscan_none", state.PortscanNone)
}

// applyMonitoring calls service_monitoring to enable or disable check execution.
// When monitoring is null/unknown (not configured), it defaults to enabled (1).
func (r *CheckResource) applyMonitoring(id string, monitoring types.Bool) error {
	val := 1
	if !monitoring.IsNull() && !monitoring.IsUnknown() && !monitoring.ValueBool() {
		val = 0
	}
	_, err := r.client.Do(client.Request{
		"command":    "service_monitoring",
		"id":         id,
		"monitoring": val,
	})
	return err
}

// applyNotification calls service_notification to enable or disable alert notifications.
// When notification is null/unknown (not configured), it defaults to enabled (1).
func (r *CheckResource) applyNotification(id string, notification types.Bool) error {
	val := 1
	if !notification.IsNull() && !notification.IsUnknown() && !notification.ValueBool() {
		val = 0
	}
	_, err := r.client.Do(client.Request{
		"command":      "service_notification",
		"id":           id,
		"notification": val,
	})
	return err
}
