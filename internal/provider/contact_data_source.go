package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
)

var _ datasource.DataSource = &ContactDataSource{}

type ContactDataSource struct {
	client *client.Client
}

type ContactDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Language types.String `tfsdk:"language"`
	Email    types.String `tfsdk:"email"`
	Mobile   types.String `tfsdk:"mobile"`
	Network  types.String `tfsdk:"network"`
	Standard types.Bool   `tfsdk:"standard"`
}

func NewContactDataSource() datasource.DataSource {
	return &ContactDataSource{}
}

func (d *ContactDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contact"
}

func (d *ContactDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Looks up an existing Serverguard24 contact by id or name. Exactly one of id or name must be set.",
		MarkdownDescription: "Looks up an existing Serverguard24 contact by `id` or `name`. Exactly one of `id` or `name` must be set.\n\n~> **Performance:** The underlying `get_contacts` endpoint ignores the `id` filter and always returns the full account; the provider filters client-side. A 30s cache with mutex-serialization prevents redundant fetches. Remaining cost is local filtering.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Contact ID. Either id or name must be set.",
				MarkdownDescription: "Contact ID. Either `id` or `name` must be set.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Contact display name. Either id or name must be set.",
				MarkdownDescription: "Contact display name. Either `id` or `name` must be set.",
			},
			"language": schema.StringAttribute{
				Computed:            true,
				Description:         "Notification language: en or de.",
				MarkdownDescription: "Notification language: `en` or `de`.",
			},
			"email": schema.StringAttribute{
				Computed:            true,
				Description:         "Email address for alert notifications.",
				MarkdownDescription: "Email address for alert notifications.",
			},
			"mobile": schema.StringAttribute{
				Computed:            true,
				Description:         "Mobile phone number for SMS alerts.",
				MarkdownDescription: "Mobile phone number for SMS alerts.",
			},
			"network": schema.StringAttribute{
				Computed:            true,
				Description:         "Network push address.",
				MarkdownDescription: "Network push address.",
			},
			"standard": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether this is the default contact.",
				MarkdownDescription: "Whether this is the default contact.",
			},
		},
	}
}

func (d *ContactDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *ContactDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ContactDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idSet := !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != ""
	nameSet := !config.Name.IsNull() && !config.Name.IsUnknown() && config.Name.ValueString() != ""
	if idSet == nameSet {
		resp.Diagnostics.AddError(
			"Invalid contact lookup",
			"Exactly one of id or name must be set.",
		)
		return
	}

	apiResp, err := d.client.ListContacts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading contacts", err.Error())
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
		resp.Diagnostics.AddError("Error parsing contact list", err.Error())
		return
	}

	matchIdx := -1
	for i, c := range contacts {
		if idSet && c.ID.String() == config.ID.ValueString() {
			matchIdx = i
			break
		}
		if nameSet && c.Name == config.Name.ValueString() {
			if matchIdx >= 0 {
				resp.Diagnostics.AddError(
					"Multiple contacts matched name",
					fmt.Sprintf("Found more than one contact named %q; use id instead.", config.Name.ValueString()),
				)
				return
			}
			matchIdx = i
		}
	}
	if matchIdx < 0 {
		key := "name=" + config.Name.ValueString()
		if idSet {
			key = "id=" + config.ID.ValueString()
		}
		resp.Diagnostics.AddError("Contact not found", fmt.Sprintf("No contact matched %s", key))
		return
	}

	c := contacts[matchIdx]
	config.ID = types.StringValue(c.ID.String())
	config.Name = types.StringValue(c.Name)
	config.Language = types.StringValue(c.Language)
	config.Email = optionalString(c.Email)
	config.Mobile = optionalString(c.Mobile)
	config.Network = optionalString(c.Network)
	config.Standard = types.BoolValue(c.Standard == 1)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// optionalString turns empty strings into null so that consumers can use
// coalesce / try() patterns to detect "not configured" fields.
func optionalString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
