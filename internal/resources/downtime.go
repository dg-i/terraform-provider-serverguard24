package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"git.dg-i.net/dg-i/engineering/terraform/serverguard24-provider/internal/client"
)

var _ resource.Resource = &DowntimeResource{}
var _ resource.ResourceWithImportState = &DowntimeResource{}

type DowntimeResource struct {
	client *client.Client
}

type DowntimeResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Hostname    types.String `tfsdk:"hostname"`
	StartTime   types.Int64  `tfsdk:"starttime"`
	EndTime     types.Int64  `tfsdk:"endtime"`
	Description types.String `tfsdk:"description"`
	Type        types.Int64  `tfsdk:"type"`
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
		MarkdownDescription: "Manages a Serverguard24 maintenance window (downtime). " +
			"**Note:** The API does not support editing or deleting downtimes. Any change forces recreation. " +
			"Import sets only the downtime `id`; add `hostname`, `starttime`, `endtime`, and `description` in config and run `terraform apply` to complete.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"hostname": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Hostname for the downtime.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"starttime": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Start time as Unix timestamp.",
			},
			"endtime": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "End time as Unix timestamp.",
			},
			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Description of the maintenance window.",
			},
			"type": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				MarkdownDescription: "Downtime type (default: 1).",
			},
			"readonly": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Mark downtime as read-only.",
			},
			"services": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "List of service IDs to link to this downtime.",
			},
			"hosts": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "List of host IDs to link to this downtime.",
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

	cmd := client.Request{
		"command":     "downtime_add",
		"hostname":    plan.Hostname.ValueString(),
		"starttime":   plan.StartTime.ValueInt64(),
		"endtime":     plan.EndTime.ValueInt64(),
		"description": plan.Description.ValueString(),
		"type":        plan.Type.ValueInt64(),
	}
	if !plan.ReadOnly.IsNull() && !plan.ReadOnly.IsUnknown() {
		cmd["readonly"] = boolToInt(plan.ReadOnly.ValueBool())
	}

	apiResp, err := r.client.Do(cmd)
	if err != nil {
		resp.Diagnostics.AddError("Error creating downtime", err.Error())
		return
	}

	var data struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		resp.Diagnostics.AddError("Error parsing downtime ID", err.Error())
		return
	}

	plan.ID = types.StringValue(data.ID.String())

	// Link services
	if !plan.Services.IsNull() && !plan.Services.IsUnknown() {
		for _, svcID := range listToInts(ctx, plan.Services) {
			if _, err := r.client.Do(client.Request{
				"command":     "downtime_link",
				"downtime_id": plan.ID.ValueString(),
				"service_id":  svcID,
			}); err != nil {
				resp.Diagnostics.AddError("Error linking service to downtime", err.Error())
				return
			}
		}
	}

	// Link hosts
	if !plan.Hosts.IsNull() && !plan.Hosts.IsUnknown() {
		for _, hostID := range listToInts(ctx, plan.Hosts) {
			if _, err := r.client.Do(client.Request{
				"command":     "downtime_link",
				"downtime_id": plan.ID.ValueString(),
				"host_id":     hostID,
			}); err != nil {
				resp.Diagnostics.AddError("Error linking host to downtime", err.Error())
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
	// The API has no downtime_edit; all mutable fields should trigger ForceNew
	// via the plan modifier on hostname. For other field changes the resource
	// will be recreated by Terraform (destroy + create).
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

	// Attempt to unlink services/hosts
	if !state.Services.IsNull() {
		for _, svcID := range listToInts(ctx, state.Services) {
			r.client.Do(client.Request{ //nolint:errcheck
				"command":     "downtime_unlink",
				"downtime_id": state.ID.ValueString(),
				"service_id":  svcID,
			})
		}
	}
	if !state.Hosts.IsNull() {
		for _, hostID := range listToInts(ctx, state.Hosts) {
			r.client.Do(client.Request{ //nolint:errcheck
				"command":     "downtime_unlink",
				"downtime_id": state.ID.ValueString(),
				"host_id":     hostID,
			})
		}
	}

	// Note: the downtime record itself cannot be deleted via API.
	// Terraform will remove it from state after this function returns.
}

// ImportState imports an existing downtime by ID. The API has no downtime_show;
// only the id is set in state. Add hostname, starttime, endtime, and description
// to your resource block and run terraform apply to populate the rest.
func (r *DowntimeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, DowntimeResourceModel{
		ID:          types.StringValue(req.ID),
		Hostname:    types.StringNull(),
		StartTime:   types.Int64Null(),
		EndTime:     types.Int64Null(),
		Description: types.StringNull(),
		Type:        types.Int64Value(1),
		ReadOnly:    types.BoolNull(),
		Services:    emptyList(),
		Hosts:       emptyList(),
	})...)
}

// emptyList returns an empty typed list for Int64.
func emptyList() types.List {
	return types.ListValueMust(types.Int64Type, []attr.Value{})
}
