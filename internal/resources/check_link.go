package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"git.dg-i.net/dg-i/engineering/terraform/serverguard24-provider/internal/client"
)

// parseLinkID splits a composite import ID "checkID/secondID" into two parts.
// Valid format: exactly one "/" with non-empty segments (e.g. "123/456").
// Returns diagnostics on invalid format (empty, no "/", multiple "/", or empty segment).
func parseLinkID(id, resourceName string) (checkID, secondID string, diags diag.Diagnostics) {
	if id == "" {
		diags.AddError("Invalid import ID", "Import ID must be in the form \"<check_id>/<second_id>\" (e.g. \"123/456\").")
		return "", "", diags
	}
	if strings.Count(id, "/") != 1 {
		diags.AddError("Invalid import ID", fmt.Sprintf("Expected format \"<check_id>/<second_id>\" for %s (exactly one \"/\"), got %q.", resourceName, id))
		return "", "", diags
	}
	parts := strings.SplitN(id, "/", 2)
	checkID = strings.TrimSpace(parts[0])
	secondID = strings.TrimSpace(parts[1])
	if checkID == "" || secondID == "" {
		diags.AddError("Invalid import ID", "Both check_id and second_id must be non-empty. Use format \"<check_id>/<second_id>\".")
		return "", "", diags
	}
	return checkID, secondID, diags
}

// CheckContactLinkResource links a check to a contact.
type CheckContactLinkResource struct {
	client *client.Client
}

type CheckContactLinkModel struct {
	ID        types.String `tfsdk:"id"`
	CheckID   types.String `tfsdk:"check_id"`
	ContactID types.String `tfsdk:"contact_id"`
}

var _ resource.ResourceWithImportState = &CheckContactLinkResource{}

func NewCheckContactLinkResource() resource.Resource {
	return &CheckContactLinkResource{}
}

func (r *CheckContactLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check_contact_link"
}

func (r *CheckContactLinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Links a monitoring check to a contact.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"check_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"contact_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *CheckContactLinkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CheckContactLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CheckContactLinkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Do(client.Request{
		"command":    "service_link_contact",
		"service_id": plan.CheckID.ValueString(),
		"contact_id": plan.ContactID.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error linking check to contact", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.CheckID.ValueString() + "/" + plan.ContactID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CheckContactLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CheckContactLinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CheckContactLinkResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Links are immutable; destroy and recreate.")
}

func (r *CheckContactLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CheckContactLinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Do(client.Request{
		"command":    "service_unlink_contact",
		"service_id": state.CheckID.ValueString(),
		"contact_id": state.ContactID.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error unlinking check from contact", err.Error())
	}
}

func (r *CheckContactLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	checkID, contactID, diags := parseLinkID(req.ID, "serverguard24_check_contact_link")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &CheckContactLinkModel{
		ID:        types.StringValue(checkID + "/" + contactID),
		CheckID:   types.StringValue(checkID),
		ContactID: types.StringValue(contactID),
	})...)
}

// CheckDistributorLinkResource links a check to a distributor.
type CheckDistributorLinkResource struct {
	client *client.Client
}

type CheckDistributorLinkModel struct {
	ID             types.String `tfsdk:"id"`
	CheckID        types.String `tfsdk:"check_id"`
	DistributorID  types.String `tfsdk:"distributor_id"`
}

var _ resource.ResourceWithImportState = &CheckDistributorLinkResource{}

func NewCheckDistributorLinkResource() resource.Resource {
	return &CheckDistributorLinkResource{}
}

func (r *CheckDistributorLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check_distributor_link"
}

func (r *CheckDistributorLinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Links a monitoring check to a distributor.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"check_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"distributor_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *CheckDistributorLinkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CheckDistributorLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CheckDistributorLinkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Do(client.Request{
		"command":        "service_link_distributor",
		"service_id":     plan.CheckID.ValueString(),
		"distributor_id": plan.DistributorID.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error linking check to distributor", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.CheckID.ValueString() + "/" + plan.DistributorID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CheckDistributorLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CheckDistributorLinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CheckDistributorLinkResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Links are immutable; destroy and recreate.")
}

func (r *CheckDistributorLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CheckDistributorLinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Do(client.Request{
		"command":        "service_unlink_distributor",
		"service_id":     state.CheckID.ValueString(),
		"distributor_id": state.DistributorID.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error unlinking check from distributor", err.Error())
	}
}

func (r *CheckDistributorLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	checkID, distributorID, diags := parseLinkID(req.ID, "serverguard24_check_distributor_link")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &CheckDistributorLinkModel{
		ID:             types.StringValue(checkID + "/" + distributorID),
		CheckID:        types.StringValue(checkID),
		DistributorID:  types.StringValue(distributorID),
	})...)
}

// CheckIntegrationLinkResource links a check to an integration.
type CheckIntegrationLinkResource struct {
	client *client.Client
}

type CheckIntegrationLinkModel struct {
	ID            types.String `tfsdk:"id"`
	CheckID       types.String `tfsdk:"check_id"`
	IntegrationID types.String `tfsdk:"integration_id"`
}

var _ resource.ResourceWithImportState = &CheckIntegrationLinkResource{}

func NewCheckIntegrationLinkResource() resource.Resource {
	return &CheckIntegrationLinkResource{}
}

func (r *CheckIntegrationLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check_integration_link"
}

func (r *CheckIntegrationLinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Links a monitoring check to an integration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"check_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"integration_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *CheckIntegrationLinkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CheckIntegrationLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CheckIntegrationLinkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Do(client.Request{
		"command":        "service_link_integration",
		"service_id":     plan.CheckID.ValueString(),
		"integration_id": plan.IntegrationID.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error linking check to integration", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.CheckID.ValueString() + "/" + plan.IntegrationID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CheckIntegrationLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CheckIntegrationLinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CheckIntegrationLinkResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Links are immutable; destroy and recreate.")
}

func (r *CheckIntegrationLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CheckIntegrationLinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Do(client.Request{
		"command":        "service_unlink_integration",
		"service_id":     state.CheckID.ValueString(),
		"integration_id": state.IntegrationID.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Error unlinking check from integration", err.Error())
	}
}

func (r *CheckIntegrationLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	checkID, integrationID, diags := parseLinkID(req.ID, "serverguard24_check_integration_link")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &CheckIntegrationLinkModel{
		ID:            types.StringValue(checkID + "/" + integrationID),
		CheckID:       types.StringValue(checkID),
		IntegrationID: types.StringValue(integrationID),
	})...)
}
