package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
)

var _ datasource.DataSource = &CheckDataSource{}

type CheckDataSource struct {
	client *client.Client
}

type CheckDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Type         types.String `tfsdk:"type"`
	Shortname    types.String `tfsdk:"shortname"`
	Description  types.String `tfsdk:"description"`
	Address      types.String `tfsdk:"address"`
	Active       types.Bool   `tfsdk:"active"`
	Notification types.Bool   `tfsdk:"notification"`
}

func NewCheckDataSource() datasource.DataSource {
	return &CheckDataSource{}
}

func (d *CheckDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check"
}

func (d *CheckDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Looks up an existing Serverguard24 check by id. Only top-level identifying fields are exposed; the type-specific params block is not surfaced.",
		MarkdownDescription: "Looks up an existing Serverguard24 check by `id`.\n\n~> **API limitation:** The API has no bulk-list endpoint for checks (`service_show` accepts only an id), so lookup by name is not supported. Only top-level identifying fields are exposed; the type-specific `params` block is not surfaced. Use the `serverguard24_check` resource if you need round-trip management.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				Description:         "Check ID.",
				MarkdownDescription: "Check ID.",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				Description:         "Check type name (e.g. http, ping, ssl_cert).",
				MarkdownDescription: "Check type name (e.g. `http`, `ping`, `ssl_cert`).",
			},
			"shortname": schema.StringAttribute{
				Computed:            true,
				Description:         "Short display name of the check.",
				MarkdownDescription: "Short display name of the check.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				Description:         "Long description of the check.",
				MarkdownDescription: "Long description of the check.",
			},
			"address": schema.StringAttribute{
				Computed:            true,
				Description:         "Address override (if any) for the check.",
				MarkdownDescription: "Address override (if any) for the check.",
			},
			"active": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether monitoring is currently active.",
				MarkdownDescription: "Whether monitoring is currently active.",
			},
			"notification": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether notifications are currently enabled.",
				MarkdownDescription: "Whether notifications are currently enabled.",
			},
		},
	}
}

func (d *CheckDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CheckDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config CheckDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := d.client.Do(ctx, client.Request{
		"command": "service_show",
		"id":      config.ID.ValueString(),
	})
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Check not found", fmt.Sprintf("No check with id=%s", config.ID.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Error reading check", err.Error())
		return
	}

	var raw map[string]any
	if err := json.Unmarshal(apiResp.Data, &raw); err != nil {
		resp.Diagnostics.AddError("Error parsing check data", err.Error())
		return
	}

	if v, ok := raw["name"].(string); ok {
		if typeName, found := checkTypeByDisplayName[strings.ToLower(v)]; found {
			config.Type = types.StringValue(typeName)
		} else {
			config.Type = types.StringValue(v)
		}
	} else {
		config.Type = types.StringNull()
	}
	config.Shortname = pickString(raw, "shortname")
	config.Description = pickString(raw, "description")
	config.Address = pickString(raw, "address")
	config.Active = pickBoolFromInt(raw, "active")
	config.Notification = pickBoolFromInt(raw, "notifications")

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func pickString(raw map[string]any, key string) types.String {
	if v, ok := raw[key].(string); ok && v != "" {
		return types.StringValue(v)
	}
	return types.StringNull()
}

func pickBoolFromInt(raw map[string]any, key string) types.Bool {
	if v, ok := raw[key].(float64); ok {
		return types.BoolValue(v == 1)
	}
	if v, ok := raw[key].(bool); ok {
		return types.BoolValue(v)
	}
	return types.BoolNull()
}
