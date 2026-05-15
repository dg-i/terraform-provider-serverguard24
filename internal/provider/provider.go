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
	const desc = "Provider for managing Serverguard24 monitoring resources."
	const mdDesc = `Provider for managing Serverguard24 monitoring resources.

## Known API limitations

The Serverguard24 JSON API has several asymmetries that affect Terraform semantics:

- **Boolean flags cannot be turned off via update.** Notification flags
  (` + "`email_*`, `sms_*`, `push_*`, `http_*`, `phone_*`" + `), ` + "`ipv6`" + `, and most
  bool flags inside the ` + "`params`" + ` block (` + "`ssl`, `check_cert`, `redirect`, `smtp_tls`" + `,
  …) are write-once. Toggling them from ` + "`true`" + ` to ` + "`false`" + ` in config has no
  effect at apply time and produces a perpetual diff. The provider emits a
  plan-time warning when it detects this case. Workaround: taint the resource
  to force recreation, or change the value in the Serverguard24 dashboard.
- **String fields cannot be cleared via update.** Setting an optional string
  attribute to ` + "`null`" + ` or omitting it after a value was previously stored will
  leave the old value in place. Same workaround.
- **Maintenance windows (` + "`serverguard24_downtime`" + `) cannot be edited or deleted
  via API.** Every attribute uses ` + "`ForceNew`" + ` and the resource is destroyed +
  recreated on change.
- **Read endpoints ignore the ID filter.** ` + "`get_hosts`/`get_contacts`" + ` return
  the full account; the provider filters client-side. Refresh performance is
  O(N×M) for accounts with many resources.
- **Error code 210 is overloaded.** It means both "invalid parameter" and
  "resource not found". The provider treats it conservatively as "gone".`

	resp.Schema = schema.Schema{
		Description:         desc,
		MarkdownDescription: mdDesc,
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
		NewHostResource,
		NewContactResource,
		NewCheckResource,
		NewDowntimeResource,
		NewCheckContactLinkResource,
		NewCheckDistributorLinkResource,
		NewCheckIntegrationLinkResource,
	}
}

func (p *Serverguard24Provider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCheckDataSource,
		NewContactDataSource,
		NewDistributorsDataSource,
		NewGroupsDataSource,
		NewHostDataSource,
		NewIntegrationsDataSource,
		NewNodesDataSource,
	}
}
