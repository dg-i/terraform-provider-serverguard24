package datasources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"git.dg-i.net/dg-i/engineering/terraform/serverguard24-provider/internal/client"
)

var _ datasource.DataSource = &DistributorsDataSource{}

type DistributorsDataSource struct {
	client *client.Client
}

type DistributorsDataSourceModel struct {
	Distributors []DistributorModel `tfsdk:"distributors"`
}

type DistributorModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewDistributorsDataSource() datasource.DataSource {
	return &DistributorsDataSource{}
}

func (d *DistributorsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_distributors"
}

func (d *DistributorsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all available Serverguard24 distributors.",
		Attributes: map[string]schema.Attribute{
			"distributors": schema.ListNestedAttribute{
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

func (d *DistributorsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DistributorsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	apiResp, err := d.client.Do(client.Request{"command": "get_distributors"})
	if err != nil {
		if client.IsPermissionError(err) {
			resp.Diagnostics.AddWarning("Insufficient permissions for get_distributors", err.Error())
			resp.Diagnostics.Append(resp.State.Set(ctx, &DistributorsDataSourceModel{Distributors: []DistributorModel{}})...)
			return
		}
		resp.Diagnostics.AddError("Error reading distributors", err.Error())
		return
	}

	var raw []map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &raw); err != nil {
		resp.Diagnostics.AddError("Error parsing distributors", err.Error())
		return
	}

	var state DistributorsDataSourceModel
	for _, item := range raw {
		id := fmt.Sprintf("%v", item["id"])
		name, _ := item["name"].(string)
		state.Distributors = append(state.Distributors, DistributorModel{
			ID:   types.StringValue(id),
			Name: types.StringValue(name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// attrTypes returns the attr.Type map for a distributor object (used in list construction).
func distributorAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
	}
}
