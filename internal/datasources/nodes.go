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
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewNodesDataSource() datasource.DataSource {
	return &NodesDataSource{}
}

func (d *NodesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nodes"
}

func (d *NodesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all available Serverguard24 monitoring nodes.",
		Attributes: map[string]schema.Attribute{
			"nodes": schema.ListNestedAttribute{
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

	var raw []map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &raw); err != nil {
		resp.Diagnostics.AddError("Error parsing nodes", err.Error())
		return
	}

	var state NodesDataSourceModel
	for _, item := range raw {
		id := fmt.Sprintf("%v", item["id"])
		name, _ := item["name"].(string)
		state.Nodes = append(state.Nodes, NodeModel{
			ID:   types.StringValue(id),
			Name: types.StringValue(name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
