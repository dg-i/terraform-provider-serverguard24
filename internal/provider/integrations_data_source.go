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

var _ datasource.DataSource = &IntegrationsDataSource{}

type IntegrationsDataSource struct {
	client *client.Client
}

type IntegrationsDataSourceModel struct {
	Integrations []IntegrationModel          `tfsdk:"integrations"`
	ByName       map[string]IntegrationModel `tfsdk:"by_name"`
	ByID         map[string]IntegrationModel `tfsdk:"by_id"`
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
			"by_name": schema.MapNestedAttribute{
				Computed:            true,
				Description:         "Map of integration name to integration object for lookup by name.",
				MarkdownDescription: "Map of integration name to integration object for lookup by name.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
			"by_id": schema.MapNestedAttribute{
				Computed:            true,
				Description:         "Map of integration ID to integration object for lookup by ID.",
				MarkdownDescription: "Map of integration ID to integration object for lookup by ID.",
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
	apiResp, err := d.client.Do(ctx, client.Request{"command": "get_integrations"})
	if err != nil {
		if client.IsPermissionError(err) {
			resp.Diagnostics.AddWarning("Insufficient permissions for get_integrations", err.Error())
			resp.Diagnostics.Append(resp.State.Set(ctx, &IntegrationsDataSourceModel{Integrations: []IntegrationModel{}, ByName: map[string]IntegrationModel{}, ByID: map[string]IntegrationModel{}})...)
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

	state := IntegrationsDataSourceModel{
		Integrations: []IntegrationModel{},
		ByName:       make(map[string]IntegrationModel, len(raw)),
		ByID:         make(map[string]IntegrationModel, len(raw)),
	}
	for _, item := range raw {
		id := fmt.Sprintf("%v", item["id"])
		name, _ := item["name"].(string)
		m := IntegrationModel{
			ID:   types.StringValue(id),
			Name: types.StringValue(name),
		}
		state.Integrations = append(state.Integrations, m)
		state.ByName[name] = m
		state.ByID[id] = m
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
