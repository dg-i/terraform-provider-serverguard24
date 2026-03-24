package datasources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"git.dg-i.net/dg-i/engineering/terraform/serverguard24-provider/internal/client"
)

var _ datasource.DataSource = &NodesDataSource{}

type NodesDataSource struct {
	client *client.Client
}

type NodesDataSourceModel struct {
	Nodes []NodeModel `tfsdk:"nodes"`
}

type NodeModel struct {
	IPv4        types.String `tfsdk:"ipv4"`
	IPv6        types.String `tfsdk:"ipv6"`
	Location    types.String `tfsdk:"location"`
	Country     types.String `tfsdk:"country"`
	CountryCode types.String `tfsdk:"country_code"`
	Region      types.String `tfsdk:"region"`
	RegionCode  types.String `tfsdk:"region_code"`
}

func NewNodesDataSource() datasource.DataSource {
	return &NodesDataSource{}
}

func (d *NodesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nodes"
}

func (d *NodesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all available Serverguard24 monitoring nodes (probe locations).",
		Attributes: map[string]schema.Attribute{
			"nodes": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ipv4":         schema.StringAttribute{Computed: true},
						"ipv6":         schema.StringAttribute{Computed: true},
						"location":     schema.StringAttribute{Computed: true},
						"country":      schema.StringAttribute{Computed: true},
						"country_code": schema.StringAttribute{Computed: true},
						"region":       schema.StringAttribute{Computed: true},
						"region_code":  schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *NodesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NodesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	apiResp, err := d.client.Do(client.Request{"command": "get_nodes"})
	if err != nil {
		if client.IsPermissionError(err) {
			resp.Diagnostics.AddWarning("Insufficient permissions for get_nodes", err.Error())
			resp.Diagnostics.Append(resp.State.Set(ctx, &NodesDataSourceModel{Nodes: []NodeModel{}})...)
			return
		}
		resp.Diagnostics.AddError("Error reading nodes", err.Error())
		return
	}

	var raw []struct {
		IPv4        string `json:"ipv4"`
		IPv6        string `json:"ipv6"`
		Location    string `json:"location"`
		Country     string `json:"country"`
		CountryCode string `json:"country_code"`
		Region      string `json:"region"`
		RegionCode  string `json:"region_code"`
	}
	if err := json.Unmarshal(apiResp.Data, &raw); err != nil {
		resp.Diagnostics.AddError("Error parsing nodes", err.Error())
		return
	}

	var state NodesDataSourceModel
	for _, item := range raw {
		state.Nodes = append(state.Nodes, NodeModel{
			IPv4:        types.StringValue(item.IPv4),
			IPv6:        types.StringValue(item.IPv6),
			Location:    types.StringValue(item.Location),
			Country:     types.StringValue(item.Country),
			CountryCode: types.StringValue(item.CountryCode),
			Region:      types.StringValue(item.Region),
			RegionCode:  types.StringValue(item.RegionCode),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
