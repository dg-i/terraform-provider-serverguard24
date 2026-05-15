package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
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

func parsePositiveID(id, fieldName string) (int64, error) {
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive numeric ID, got %q", fieldName, id)
	}
	return parsed, nil
}

type positiveIDStringValidator struct{}

func (v positiveIDStringValidator) Description(_ context.Context) string {
	return "must be a positive numeric ID"
}

func (v positiveIDStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v positiveIDStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := parsePositiveID(req.ConfigValue.ValueString(), req.Path.String()); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid ID",
			err.Error(),
		)
	}
}

// linkResource is a generic implementation for check link resources.
// All three link types (contact, distributor, integration) are structurally
// identical and differ only in the second field name and API commands.
type linkResource struct {
	client      *client.Client
	typeSuffix  string // e.g. "_check_contact_link"
	secondField string // e.g. "contact_id"
	linkCmd     string // e.g. "service_link_contact"
	unlinkCmd   string // e.g. "service_unlink_contact"
	description string
}

var _ resource.ResourceWithImportState = &linkResource{}

func (r *linkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.typeSuffix
}

func (r *linkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         r.description,
		MarkdownDescription: r.description,
		Version:             0,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Composite ID in the form check_id/second_id.",
				MarkdownDescription: "Composite ID in the form `check_id/second_id`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"check_id": schema.StringAttribute{
				Required:            true,
				Description:         "ID of the monitoring check.",
				MarkdownDescription: "ID of the monitoring check.",
				Validators:          []validator.String{positiveIDStringValidator{}},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			r.secondField: schema.StringAttribute{
				Required:            true,
				Description:         "ID of the linked resource.",
				MarkdownDescription: "ID of the linked resource.",
				Validators:          []validator.String{positiveIDStringValidator{}},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *linkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *linkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var checkID, secondID types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("check_id"), &checkID)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(r.secondField), &secondID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svcIDInt, err := parsePositiveID(checkID.ValueString(), "check_id")
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("check_id"), "Invalid check_id", err.Error())
		return
	}
	secondIDInt, err := parsePositiveID(secondID.ValueString(), r.secondField)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root(r.secondField), "Invalid "+r.secondField, err.Error())
		return
	}
	if _, err := r.client.Do(ctx, client.Request{
		"command":     r.linkCmd,
		"id":          svcIDInt,
		r.secondField: secondIDInt,
	}); err != nil {
		resp.Diagnostics.AddError("Error creating link", err.Error())
		return
	}

	id := types.StringValue(checkID.ValueString() + "/" + secondID.ValueString())
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("check_id"), checkID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(r.secondField), secondID)...)
}

func (r *linkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// No read API exists for links; pass state through unchanged.
	var id, checkID, secondID types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("check_id"), &checkID)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(r.secondField), &secondID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("check_id"), checkID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(r.secondField), secondID)...)
}

func (r *linkResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Links are immutable; destroy and recreate.")
}

func (r *linkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var checkID, secondID types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("check_id"), &checkID)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(r.secondField), &secondID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svcIDInt, err := parsePositiveID(checkID.ValueString(), "check_id")
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("check_id"), "Invalid check_id", err.Error())
		return
	}
	secondIDInt, err := parsePositiveID(secondID.ValueString(), r.secondField)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root(r.secondField), "Invalid "+r.secondField, err.Error())
		return
	}
	if _, err := r.client.Do(ctx, client.Request{
		"command":     r.unlinkCmd,
		"id":          svcIDInt,
		r.secondField: secondIDInt,
	}); err != nil {
		resp.Diagnostics.AddError("Error deleting link", err.Error())
	}
}

func (r *linkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	checkID, secondID, diags := parseLinkID(req.ID, "serverguard24"+r.typeSuffix)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := checkID + "/" + secondID
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(id))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("check_id"), types.StringValue(checkID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(r.secondField), types.StringValue(secondID))...)
}

func NewCheckContactLinkResource() resource.Resource {
	return &linkResource{
		typeSuffix:  "_check_contact_link",
		secondField: "contact_id",
		linkCmd:     "service_link_contact",
		unlinkCmd:   "service_unlink_contact",
		description: "Links a monitoring check to a contact.",
	}
}

func NewCheckDistributorLinkResource() resource.Resource {
	return &linkResource{
		typeSuffix:  "_check_distributor_link",
		secondField: "distributor_id",
		linkCmd:     "service_link_distributor",
		unlinkCmd:   "service_unlink_distributor",
		description: "Links a monitoring check to a distributor.",
	}
}

func NewCheckIntegrationLinkResource() resource.Resource {
	return &linkResource{
		typeSuffix:  "_check_integration_link",
		secondField: "integration_id",
		linkCmd:     "service_link_integration",
		unlinkCmd:   "service_unlink_integration",
		description: "Links a monitoring check to an integration.",
	}
}
