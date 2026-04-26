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

var _ datasource.DataSource = &GroupsDataSource{}

type GroupsDataSource struct {
	client *client.Client
}

type GroupsDataSourceModel struct {
	Groups []GroupModel          `tfsdk:"groups"`
	ByName map[string]GroupModel `tfsdk:"by_name"`
	ByID   map[string]GroupModel `tfsdk:"by_id"`
}

type GroupModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewGroupsDataSource() datasource.DataSource {
	return &GroupsDataSource{}
}

func (d *GroupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_groups"
}

func (d *GroupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Retrieves all available Serverguard24 host groups. NOTE: The groups feature appears to have no UI and no API commands to create or manage groups. Status is unknown — get_groups consistently returns an empty list on all known accounts.",
		MarkdownDescription: "Retrieves all available Serverguard24 host groups.\n\n~> **Status unknown:** The groups feature appears to have no UI and no API commands to create or manage groups. `get_groups` consistently returns an empty list on all known accounts. This data source is kept for completeness but may not be functional.",
		Attributes: map[string]schema.Attribute{
			"groups": schema.ListNestedAttribute{
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
				Description:         "Map of group name to group object for lookup by name.",
				MarkdownDescription: "Map of group name to group object for lookup by name.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
			"by_id": schema.MapNestedAttribute{
				Computed:            true,
				Description:         "Map of group ID to group object for lookup by ID.",
				MarkdownDescription: "Map of group ID to group object for lookup by ID.",
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

func (d *GroupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GroupsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	apiResp, err := d.client.Do(client.Request{"command": "get_groups"})
	if err != nil {
		if client.IsPermissionError(err) {
			resp.Diagnostics.AddWarning("Insufficient permissions for get_groups", err.Error())
			resp.Diagnostics.Append(resp.State.Set(ctx, &GroupsDataSourceModel{Groups: []GroupModel{}, ByName: map[string]GroupModel{}, ByID: map[string]GroupModel{}})...)
			return
		}
		resp.Diagnostics.AddError("Error reading groups", err.Error())
		return
	}

	var raw []map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &raw); err != nil {
		resp.Diagnostics.AddError("Error parsing groups", err.Error())
		return
	}

	state := GroupsDataSourceModel{
		Groups: []GroupModel{},
		ByName: make(map[string]GroupModel, len(raw)),
		ByID:   make(map[string]GroupModel, len(raw)),
	}
	for _, item := range raw {
		id := fmt.Sprintf("%v", item["id"])
		name, _ := item["name"].(string)
		m := GroupModel{
			ID:   types.StringValue(id),
			Name: types.StringValue(name),
		}
		state.Groups = append(state.Groups, m)
		state.ByName[name] = m
		state.ByID[id] = m
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
