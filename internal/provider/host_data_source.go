package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
)

var _ datasource.DataSource = &HostDataSource{}

type HostDataSource struct {
	client *client.Client
}

type HostDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Address types.String `tfsdk:"address"`
	Groups  types.List   `tfsdk:"groups"`
}

func NewHostDataSource() datasource.DataSource {
	return &HostDataSource{}
}

func (d *HostDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host"
}

func (d *HostDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Looks up an existing Serverguard24 host by id or name. Exactly one of id or name must be set.",
		MarkdownDescription: "Looks up an existing Serverguard24 host by `id` or `name`. Exactly one of `id` or `name` must be set.\n\n~> **Performance:** The underlying `get_hosts` endpoint ignores the `id` filter and always returns the full account; the provider filters client-side. A 30s cache with mutex-serialization prevents redundant fetches. Remaining cost is local filtering.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Host ID. Either id or name must be set.",
				MarkdownDescription: "Host ID. Either `id` or `name` must be set.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Host name (alias). Either id or name must be set.",
				MarkdownDescription: "Host name (alias). Either `id` or `name` must be set.",
			},
			"address": schema.StringAttribute{
				Computed:            true,
				Description:         "IP address or FQDN of the host.",
				MarkdownDescription: "IP address or FQDN of the host.",
			},
			"groups": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.Int64Type,
				Description:         "List of group IDs the host is assigned to.",
				MarkdownDescription: "List of group IDs the host is assigned to.",
			},
		},
	}
}

func (d *HostDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *HostDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config HostDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idSet := !config.ID.IsNull() && !config.ID.IsUnknown() && config.ID.ValueString() != ""
	nameSet := !config.Name.IsNull() && !config.Name.IsUnknown() && config.Name.ValueString() != ""
	if idSet == nameSet {
		resp.Diagnostics.AddError(
			"Invalid host lookup",
			"Exactly one of id or name must be set.",
		)
		return
	}

	apiResp, err := d.client.ListHosts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading hosts", err.Error())
		return
	}

	var hosts []struct {
		ID      json.Number `json:"id"`
		Alias   string      `json:"alias"`
		Address string      `json:"address"`
		Groups  []int64     `json:"groups"`
	}
	if err := json.Unmarshal(apiResp.Data, &hosts); err != nil {
		resp.Diagnostics.AddError("Error parsing host list", err.Error())
		return
	}

	matchIdx := -1
	for i, h := range hosts {
		if idSet && h.ID.String() == config.ID.ValueString() {
			matchIdx = i
			break
		}
		if nameSet && h.Alias == config.Name.ValueString() {
			if matchIdx >= 0 {
				resp.Diagnostics.AddError(
					"Multiple hosts matched name",
					fmt.Sprintf("Found more than one host named %q; use id instead.", config.Name.ValueString()),
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
		resp.Diagnostics.AddError("Host not found", fmt.Sprintf("No host matched %s", key))
		return
	}

	h := hosts[matchIdx]
	config.ID = types.StringValue(h.ID.String())
	config.Name = types.StringValue(h.Alias)
	config.Address = types.StringValue(h.Address)
	if len(h.Groups) > 0 {
		elems := make([]attr.Value, len(h.Groups))
		for i, g := range h.Groups {
			elems[i] = types.Int64Value(g)
		}
		list, diags := types.ListValue(types.Int64Type, elems)
		resp.Diagnostics.Append(diags...)
		config.Groups = list
	} else {
		config.Groups = types.ListNull(types.Int64Type)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
