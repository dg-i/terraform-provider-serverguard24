package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
	"github.com/dg-i/terraform-provider-serverguard24/internal/datasources"
	"github.com/dg-i/terraform-provider-serverguard24/internal/resources"
)

var _ provider.Provider = &Serverguard24Provider{}

type Serverguard24Provider struct {
	version string
}

type Serverguard24ProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &Serverguard24Provider{version: version}
	}
}

func (p *Serverguard24Provider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "serverguard24"
	resp.Version = p.version
}

func (p *Serverguard24Provider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Provider for managing Serverguard24 monitoring resources.",
		MarkdownDescription: "Provider for managing Serverguard24 monitoring resources.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Description:         "Serverguard24 API token. Can also be set via the SG24_API_TOKEN environment variable.",
				MarkdownDescription: "Serverguard24 API token. Can also be set via the `SG24_API_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *Serverguard24Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config Serverguard24ProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token := os.Getenv("SG24_API_TOKEN")
	if !config.APIToken.IsNull() && !config.APIToken.IsUnknown() {
		token = config.APIToken.ValueString()
	}

	if token == "" {
		resp.Diagnostics.AddError(
			"Missing API token",
			"Set api_token in the provider block or the SG24_API_TOKEN environment variable.",
		)
		return
	}

	c := client.New(token, p.version)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *Serverguard24Provider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewHostResource,
		resources.NewContactResource,
		resources.NewCheckResource,
		resources.NewDowntimeResource,
		resources.NewCheckContactLinkResource,
		resources.NewCheckDistributorLinkResource,
		resources.NewCheckIntegrationLinkResource,
	}
}

func (p *Serverguard24Provider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewDistributorsDataSource,
		datasources.NewGroupsDataSource,
		datasources.NewIntegrationsDataSource,
		datasources.NewNodesDataSource,
	}
}
