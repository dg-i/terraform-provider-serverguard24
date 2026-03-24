package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"git.dg-i.net/dg-i/engineering/terraform/serverguard24-provider/internal/client"
)

var _ resource.Resource = &ContactResource{}
var _ resource.ResourceWithImportState = &ContactResource{}

type ContactResource struct {
	client *client.Client
}

type ContactResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Language types.String `tfsdk:"language"`
	Email    types.String `tfsdk:"email"`
	Mobile   types.String `tfsdk:"mobile"`
	Network  types.String `tfsdk:"network"`
	Standard types.Bool   `tfsdk:"standard"`
}

func NewContactResource() resource.Resource {
	return &ContactResource{}
}

func (r *ContactResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contact"
}

func (r *ContactResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Serverguard24 contact.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"language": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Notification language: `en` or `de`.",
			},
			"email": schema.StringAttribute{
				Optional: true,
			},
			"mobile": schema.StringAttribute{
				Optional: true,
			},
			"network": schema.StringAttribute{
				Optional: true,
			},
			"standard": schema.BoolAttribute{
				Optional: true,
			},
		},
	}
}

func (r *ContactResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ContactResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ContactResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := client.Request{
		"command":  "contact_add",
		"name":     plan.Name.ValueString(),
		"language": plan.Language.ValueString(),
	}
	setOptionalString(cmd, "email", plan.Email)
	setOptionalString(cmd, "mobile", plan.Mobile)
	setOptionalString(cmd, "network", plan.Network)
	if !plan.Standard.IsNull() && !plan.Standard.IsUnknown() {
		cmd["standard"] = boolToInt(plan.Standard.ValueBool())
	}

	apiResp, err := r.client.Do(cmd)
	if err != nil {
		resp.Diagnostics.AddError("Error creating contact", err.Error())
		return
	}

	var data struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		resp.Diagnostics.AddError("Error parsing contact ID", err.Error())
		return
	}

	plan.ID = types.StringValue(data.ID.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ContactResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ContactResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.Do(client.Request{
		"command": "get_contacts",
		"id":      state.ID.ValueString(),
	})
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading contact", err.Error())
		return
	}

	type contactItem struct {
		ID       json.Number `json:"id"`
		Name     string      `json:"name"`
		Language string      `json:"language"`
		Email    string      `json:"email"`
		Mobile   string      `json:"mobile"`
		Network  string      `json:"network"`
		Standard int         `json:"standard"`
	}
	var contacts []contactItem
	if err := json.Unmarshal(apiResp.Data, &contacts); err != nil {
		resp.Diagnostics.AddError("Error parsing contact data", err.Error())
		return
	}
	// The API returns all contacts regardless of the id filter; find the matching one.
	var c *contactItem
	for i := range contacts {
		if contacts[i].ID.String() == state.ID.ValueString() {
			c = &contacts[i]
			break
		}
	}
	if c == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.Name = types.StringValue(c.Name)
	state.Language = types.StringValue(c.Language)
	state.Email = nullableString(c.Email)
	state.Mobile = nullableString(c.Mobile)
	state.Network = nullableString(c.Network)
	state.Standard = types.BoolValue(c.Standard == 1)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ContactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ContactResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ContactResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := client.Request{
		"command":  "contact_edit",
		"id":       state.ID.ValueString(),
		"name":     plan.Name.ValueString(),
		"language": plan.Language.ValueString(),
	}
	setOptionalString(cmd, "email", plan.Email)
	setOptionalString(cmd, "mobile", plan.Mobile)
	setOptionalString(cmd, "network", plan.Network)
	if !plan.Standard.IsNull() && !plan.Standard.IsUnknown() {
		cmd["standard"] = boolToInt(plan.Standard.ValueBool())
	}

	if _, err := r.client.Do(cmd); err != nil {
		resp.Diagnostics.AddError("Error updating contact", err.Error())
		return
	}

	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ContactResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ContactResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Do(client.Request{
		"command": "contact_delete",
		"id":      state.ID.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error deleting contact", err.Error())
	}
}

func (r *ContactResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, ContactResourceModel{
		ID: types.StringValue(req.ID),
	})...)
}

// helpers

func setOptionalString(cmd client.Request, key string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() {
		cmd[key] = v.ValueString()
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// nullableString returns types.StringNull() when s is empty, so that optional
// fields the API did not set do not produce a perpetual diff against null config.
func nullableString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
