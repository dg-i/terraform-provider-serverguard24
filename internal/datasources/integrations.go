package datasources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
)

var _ datasource.DataSource = &IntegrationsDataSource{}

type IntegrationsDataSource struct {
	client *client.Client
}

type IntegrationsDataSourceModel struct {
	Integrations []IntegrationModel `tfsdk:"integrations"`
}

type IntegrationModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewIntegrationsDataSource() datasource.DataSource {
	return &IntegrationsDataSource{}
}

func (d *IntegrationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integrations"
}

func (d *IntegrationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves all available Serverguard24 integrations.",
		MarkdownDescription: "Retrieves all available Serverguard24 integrations.",
		Attributes: map[string]schema.Attribute{
			"integrations": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *IntegrationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IntegrationsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	apiResp, err := d.client.Do(client.Request{"command": "get_integrations"})
	if err != nil {
		if client.IsPermissionError(err) {
			resp.Diagnostics.AddWarning("Insufficient permissions for get_integrations", err.Error())
			resp.Diagnostics.Append(resp.State.Set(ctx, &IntegrationsDataSourceModel{Integrations: []IntegrationModel{}})...)
			return
		}
		resp.Diagnostics.AddError("Error reading integrations", err.Error())
		return
	}

	var raw []map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &raw); err != nil {
		resp.Diagnostics.AddError("Error parsing integrations", err.Error())
		return
	}

	var state IntegrationsDataSourceModel
	for _, item := range raw {
		id := fmt.Sprintf("%v", item["id"])
		name, _ := item["name"].(string)
		state.Integrations = append(state.Integrations, IntegrationModel{
			ID:   types.StringValue(id),
			Name: types.StringValue(name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
