package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
)

var _ resource.Resource = &HostResource{}
var _ resource.ResourceWithImportState = &HostResource{}

type HostResource struct {
	client *client.Client
}

type HostResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Address types.String `tfsdk:"address"`
	Groups  types.List   `tfsdk:"groups"`
}

func NewHostResource() resource.Resource {
	return &HostResource{}
}

func (r *HostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host"
}

func (r *HostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a Serverguard24 host.",
		MarkdownDescription: "Manages a Serverguard24 host.",
		Version:             0,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Host ID.",
				MarkdownDescription: "Host ID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Hostname or display name.",
				MarkdownDescription: "Hostname or display name.",
			},
			"address": schema.StringAttribute{
				Required:            true,
				Description:         "IP address or FQDN of the host.",
				MarkdownDescription: "IP address or FQDN of the host.",
			},
			"groups": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.Int64Type,
				Description:         "List of group IDs to assign this host to. NOTE: The groups feature appears to have no UI and no API commands to create groups. Status is unknown — this attribute is accepted by the API but group IDs cannot be obtained in practice.",
				MarkdownDescription: "List of group IDs to assign this host to.\n\n~> **Status unknown:** The groups feature appears to have no UI and no API commands to create or manage groups. This attribute is accepted by the API but group IDs cannot be obtained in practice.",
			},
		},
	}
}

func (r *HostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := client.Request{
		"command": "host_add",
		"name":    plan.Name.ValueString(),
		"address": plan.Address.ValueString(),
	}
	if !plan.Groups.IsNull() && !plan.Groups.IsUnknown() {
		cmd["groups"] = listToInts(ctx, plan.Groups)
	}

	apiResp, err := r.client.Do(cmd)
	if err != nil {
		resp.Diagnostics.AddError("Error creating host", err.Error())
		return
	}

	var data struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		resp.Diagnostics.AddError("Error parsing host ID", err.Error())
		return
	}

	plan.ID = types.StringValue(data.ID.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.Do(client.Request{
		"command": "get_hosts",
		"id":      state.ID.ValueString(),
	})
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading host", err.Error())
		return
	}

	var hosts []struct {
		ID      json.Number `json:"id"`
		Name    string      `json:"alias"` // API returns name field as "alias"
		Address string      `json:"address"`
		Groups  []int64     `json:"groups"`
	}
	if err := json.Unmarshal(apiResp.Data, &hosts); err != nil {
		resp.Diagnostics.AddError("Error parsing host data", err.Error())
		return
	}
	// The API returns all hosts regardless of the id filter; find the matching one.
	var h *struct {
		ID      json.Number `json:"id"`
		Name    string      `json:"alias"` // API returns name field as "alias"
		Address string      `json:"address"`
		Groups  []int64     `json:"groups"`
	}
	for i := range hosts {
		if hosts[i].ID.String() == state.ID.ValueString() {
			h = &hosts[i]
			break
		}
	}
	if h == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.Name = types.StringValue(h.Name)
	state.Address = types.StringValue(h.Address)
	if len(h.Groups) > 0 {
		elems := make([]attr.Value, len(h.Groups))
		for i, g := range h.Groups {
			elems[i] = types.Int64Value(g)
		}
		list, diags := types.ListValue(types.Int64Type, elems)
		resp.Diagnostics.Append(diags...)
		state.Groups = list
	} else {
		// API returned empty groups – preserve null if config omits groups,
		// but ensure the list type is always correct.
		state.Groups = types.ListNull(types.Int64Type)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan HostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state HostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := client.Request{
		"command": "host_edit",
		"id":      state.ID.ValueString(),
		"name":    plan.Name.ValueString(),
		"address": plan.Address.ValueString(),
	}
	if !plan.Groups.IsNull() && !plan.Groups.IsUnknown() {
		cmd["groups"] = listToInts(ctx, plan.Groups)
	}

	if _, err := r.client.Do(cmd); err != nil {
		resp.Diagnostics.AddError("Error updating host", err.Error())
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Do(client.Request{
		"command": "host_delete",
		"id":      state.ID.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error deleting host", err.Error())
	}
}

func (r *HostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, HostResourceModel{
		ID:     types.StringValue(req.ID),
		Groups: types.ListNull(types.Int64Type),
	})...)
}

func listToInts(ctx context.Context, l types.List) []int64 {
	var elems []types.Int64
	l.ElementsAs(ctx, &elems, false)
	out := make([]int64, len(elems))
	for i, e := range elems {
		out[i] = e.ValueInt64()
	}
	return out
}
