package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
)

// downtimeTypeByName maps human-readable downtime type names to API integer IDs.
// The API currently only documents type 1; additional types may be added in future.
var downtimeTypeByName = map[string]int64{
	"maintenance": 1,
}

var _ resource.Resource = &DowntimeResource{}
var _ resource.ResourceWithImportState = &DowntimeResource{}

type numericStringListValidator struct{}

func (v numericStringListValidator) Description(_ context.Context) string {
	return "all elements must be positive numeric IDs"
}

func (v numericStringListValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v numericStringListValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var elems []types.String
	resp.Diagnostics.Append(req.ConfigValue.ElementsAs(ctx, &elems, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for i, elem := range elems {
		if elem.IsNull() || elem.IsUnknown() {
			continue
		}
		if _, err := parsePositiveID(elem.ValueString(), req.Path.String()); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtListIndex(i),
				"Invalid ID",
				err.Error(),
			)
		}
	}
}

type DowntimeResource struct {
	client *client.Client
}

type DowntimeResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Hostname    types.String `tfsdk:"hostname"`
	StartTime   types.String `tfsdk:"start_time"`
	EndTime     types.String `tfsdk:"end_time"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	ReadOnly    types.Bool   `tfsdk:"readonly"`
	Services    types.List   `tfsdk:"services"`
	Hosts       types.List   `tfsdk:"hosts"`
}

func NewDowntimeResource() resource.Resource {
	return &DowntimeResource{}
}

func (r *DowntimeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_downtime"
}

func (r *DowntimeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Serverguard24 maintenance window (downtime). " +
			"Note: The API does not support editing or deleting downtimes. Any change forces recreation. " +
			"Import sets only the downtime id; add all other attributes to config and run terraform apply to complete.",
		MarkdownDescription: "Manages a Serverguard24 maintenance window (downtime). " +
			"**Note:** The API does not support editing or deleting downtimes. Any change forces recreation. " +
			"Import sets only the downtime `id`; add all other attributes to config and run `terraform apply` to complete.",
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"hostname": schema.StringAttribute{
				Required:            true,
				Description:         "Name of the host this maintenance window applies to.",
				MarkdownDescription: "Name of the host this maintenance window applies to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"start_time": schema.StringAttribute{
				Required:            true,
				Description:         "Start of the maintenance window in RFC 3339 format (e.g. 2026-04-01T00:00:00Z).",
				MarkdownDescription: "Start of the maintenance window in RFC 3339 format (e.g. `2026-04-01T00:00:00Z`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"end_time": schema.StringAttribute{
				Required:            true,
				Description:         "End of the maintenance window in RFC 3339 format (e.g. 2026-04-01T02:00:00Z). Maximum duration is 23:59:59.",
				MarkdownDescription: "End of the maintenance window in RFC 3339 format (e.g. `2026-04-01T02:00:00Z`). Maximum duration is 23:59:59.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				Required:            true,
				Description:         "Description of the maintenance window.",
				MarkdownDescription: "Description of the maintenance window.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("maintenance"),
				Description:         "Maintenance window type. Currently only maintenance is supported.",
				MarkdownDescription: "Maintenance window type. Currently only `maintenance` is supported.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"readonly": schema.BoolAttribute{
				Optional:            true,
				Description:         "Mark the maintenance window as read-only in the Serverguard24 UI.",
				MarkdownDescription: "Mark the maintenance window as read-only in the Serverguard24 UI.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"services": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				Description:         "List of check IDs (as strings) to suppress alerts for during this maintenance window.",
				MarkdownDescription: "List of check IDs to suppress alerts for during this maintenance window. Accepts string IDs so you can reference `serverguard24_check.example.id` directly.",
				Validators:          []validator.List{numericStringListValidator{}},
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"hosts": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				Description:         "List of host IDs (as strings) to suppress alerts for during this maintenance window.",
				MarkdownDescription: "List of host IDs to suppress alerts for during this maintenance window. Accepts string IDs so you can reference `serverguard24_host.example.id` directly.",
				Validators:          []validator.List{numericStringListValidator{}},
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *DowntimeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DowntimeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DowntimeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	startUnix, err := parseRFC3339Unix(plan.StartTime.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid start_time", err.Error())
		return
	}
	endUnix, err := parseRFC3339Unix(plan.EndTime.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid end_time", err.Error())
		return
	}

	typeID, ok := downtimeTypeByName[plan.Type.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Invalid type", fmt.Sprintf("%q is not a valid downtime type", plan.Type.ValueString()))
		return
	}

	cmd := client.Request{
		"command":     "downtime_add",
		"hostname":    plan.Hostname.ValueString(),
		"starttime":   startUnix,
		"endtime":     endUnix,
		"description": plan.Description.ValueString(),
		"type":        typeID,
	}
	if !plan.ReadOnly.IsNull() && !plan.ReadOnly.IsUnknown() {
		cmd["readonly"] = boolToInt(plan.ReadOnly.ValueBool())
	}

	apiResp, err := r.client.Do(cmd)
	if err != nil {
		if client.IsAlreadyExists(err) {
			// The API has no delete, so a previously-untracked downtime with the
			// same params may still exist. Look it up by description to adopt it.
			id, lookupErr := findDowntimeByDescription(r.client, plan.Description.ValueString())
			if lookupErr != nil {
				resp.Diagnostics.AddError("Downtime already exists and lookup failed", lookupErr.Error())
				return
			}
			if id == "" {
				resp.Diagnostics.AddError("Downtime already exists", "Could not locate the existing downtime by description. Remove it manually in the Serverguard24 UI.")
				return
			}
			plan.ID = types.StringValue(id)
		} else {
			resp.Diagnostics.AddError("Error creating downtime", err.Error())
			return
		}
	} else {
		var data struct {
			ID json.Number `json:"dt_id"` // downtime_add returns "dt_id", not "id"
		}
		if err := json.Unmarshal(apiResp.Data, &data); err != nil {
			resp.Diagnostics.AddError("Error parsing downtime ID", err.Error())
			return
		}
		plan.ID = types.StringValue(data.ID.String())
	}

	// Link services and hosts in a single call each (API accepts arrays).
	if !plan.Services.IsNull() && !plan.Services.IsUnknown() {
		services, invalid := stringListToInt64s(ctx, plan.Services)
		if len(invalid) > 0 {
			resp.Diagnostics.AddError("Invalid services", invalidIDMessage(invalid))
			return
		}
		if len(services) > 0 {
			if _, err := r.client.Do(client.Request{
				"command":  "downtime_link",
				"id":       plan.ID.ValueString(),
				"services": services,
			}); err != nil {
				resp.Diagnostics.AddError("Error linking services to downtime", err.Error())
				return
			}
		}
	}

	if !plan.Hosts.IsNull() && !plan.Hosts.IsUnknown() {
		hosts, invalid := stringListToInt64s(ctx, plan.Hosts)
		if len(invalid) > 0 {
			resp.Diagnostics.AddError("Invalid hosts", invalidIDMessage(invalid))
			return
		}
		if len(hosts) > 0 {
			if _, err := r.client.Do(client.Request{
				"command": "downtime_link",
				"id":      plan.ID.ValueString(),
				"hosts":   hosts,
			}); err != nil {
				resp.Diagnostics.AddError("Error linking hosts to downtime", err.Error())
				return
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DowntimeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// The API has no downtime_show command; we store the state as-is.
	var state DowntimeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DowntimeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All mutable fields have RequiresReplace, so Terraform should never call
	// Update. If it does, surface an error rather than silently succeeding.
	resp.Diagnostics.AddError(
		"Update not supported",
		"Serverguard24 does not support editing downtimes. Please destroy and recreate.",
	)
}

func (r *DowntimeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The API has no downtime_delete. We unlink and warn the user.
	// Terraform will remove the resource from state regardless.
	var state DowntimeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Attempt to unlink services/hosts. The downtime record itself cannot be
	// deleted via the API; Terraform removes it from state after this returns.
	if !state.Services.IsNull() && !state.Services.IsUnknown() {
		services, invalid := stringListToInt64s(ctx, state.Services)
		if len(invalid) > 0 {
			resp.Diagnostics.AddWarning("Invalid services in state", invalidIDMessage(invalid))
		}
		if len(services) > 0 {
			if _, err := r.client.Do(client.Request{
				"command":  "downtime_unlink",
				"id":       state.ID.ValueString(),
				"services": services,
			}); err != nil {
				resp.Diagnostics.AddWarning(
					"Failed to unlink services from downtime",
					fmt.Sprintf("%s — the links may persist in Serverguard24.", err),
				)
			}
		}
	}

	if !state.Hosts.IsNull() && !state.Hosts.IsUnknown() {
		hosts, invalid := stringListToInt64s(ctx, state.Hosts)
		if len(invalid) > 0 {
			resp.Diagnostics.AddWarning("Invalid hosts in state", invalidIDMessage(invalid))
		}
		if len(hosts) > 0 {
			if _, err := r.client.Do(client.Request{
				"command": "downtime_unlink",
				"id":      state.ID.ValueString(),
				"hosts":   hosts,
			}); err != nil {
				resp.Diagnostics.AddWarning(
					"Failed to unlink hosts from downtime",
					fmt.Sprintf("%s — the links may persist in Serverguard24.", err),
				)
			}
		}
	}
}

// ImportState imports an existing downtime by ID. The API has no downtime_show;
// only the id is set in state. Add hostname, start_time, end_time, and description
// to your resource block and run terraform apply to populate the rest.
func (r *DowntimeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, DowntimeResourceModel{
		ID:          types.StringValue(req.ID),
		Hostname:    types.StringNull(),
		StartTime:   types.StringNull(),
		EndTime:     types.StringNull(),
		Description: types.StringNull(),
		Type:        types.StringValue("maintenance"),
		ReadOnly:    types.BoolNull(),
		Services:    emptyList(),
		Hosts:       emptyList(),
	})...)
}

// emptyList returns an empty typed list for String (used by services/hosts).
func emptyList() types.List {
	return types.ListValueMust(types.StringType, []attr.Value{})
}

// stringListToInt64s converts a types.List of strings to []int64 for API calls.
// Invalid entries are returned separately so callers can reject or warn.
func stringListToInt64s(ctx context.Context, l types.List) ([]int64, []string) {
	var elems []types.String
	l.ElementsAs(ctx, &elems, false)
	out := make([]int64, 0, len(elems))
	invalid := make([]string, 0)
	for _, e := range elems {
		if e.IsNull() || e.IsUnknown() {
			continue
		}
		v, err := parsePositiveID(e.ValueString(), "id")
		if err != nil {
			invalid = append(invalid, e.ValueString())
			continue
		}
		out = append(out, v)
	}
	return out, invalid
}

func invalidIDMessage(ids []string) string {
	return fmt.Sprintf("All IDs must be positive numeric strings. Invalid values: %q", ids)
}

// parseRFC3339Unix parses an RFC 3339 timestamp string and returns its Unix epoch value.
func parseRFC3339Unix(s string) (int64, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0, fmt.Errorf("expected RFC 3339 format (e.g. 2026-04-01T00:00:00Z), got %q: %w", s, err)
	}
	return t.Unix(), nil
}

// findDowntimeByDescription looks up all downtimes and returns the ID of the
// first one whose description matches. Returns "" if none match.
func findDowntimeByDescription(c *client.Client, description string) (string, error) {
	apiResp, err := c.Do(client.Request{"command": "get_downtimes"})
	if err != nil {
		return "", fmt.Errorf("get_downtimes: %w", err)
	}
	var items []struct {
		ID          json.Number `json:"id"`
		Description string      `json:"description"`
	}
	if err := json.Unmarshal(apiResp.Data, &items); err != nil {
		return "", fmt.Errorf("parsing get_downtimes response: %w", err)
	}
	for _, item := range items {
		if item.Description == description {
			return item.ID.String(), nil
		}
	}
	return "", nil
}
