package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
)

func TestProvider_Configure_TokenFromConfig(t *testing.T) {
	ctx := context.Background()
	p := New("test")()
	var schemaResp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)

	rawVal := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{"api_token": tftypes.String}},
		map[string]tftypes.Value{"api_token": tftypes.NewValue(tftypes.String, "config-token")},
	)
	req := provider.ConfigureRequest{
		Config: tfsdk.Config{Raw: rawVal, Schema: schemaResp.Schema},
	}
	var resp provider.ConfigureResponse
	p.Configure(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure() diagnostics: %s", resp.Diagnostics)
	}
	if resp.ResourceData == nil {
		t.Error("Configure() ResourceData should be set")
	}
	if resp.DataSourceData == nil {
		t.Error("Configure() DataSourceData should be set")
	}
	if c, ok := resp.ResourceData.(*client.Client); !ok || c == nil {
		t.Error("ResourceData should be *client.Client")
	}
}

func TestProvider_Configure_TokenFromEnv(t *testing.T) {
	const envToken = "env-token-value"
	if err := os.Setenv("SG24_API_TOKEN", envToken); err != nil {
		t.Skip("Cannot set env:", err)
	}
	defer os.Unsetenv("SG24_API_TOKEN")

	ctx := context.Background()
	p := New("test")()
	var schemaResp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)

	// Empty config: api_token null
	rawVal := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{"api_token": tftypes.String}},
		map[string]tftypes.Value{"api_token": tftypes.NewValue(tftypes.String, nil)},
	)
	req := provider.ConfigureRequest{
		Config: tfsdk.Config{Raw: rawVal, Schema: schemaResp.Schema},
	}
	var resp provider.ConfigureResponse
	p.Configure(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure() diagnostics: %s", resp.Diagnostics)
	}
	if resp.ResourceData == nil {
		t.Error("Configure() ResourceData should be set when token from env")
	}
}

func TestProvider_Configure_MissingToken(t *testing.T) {
	os.Unsetenv("SG24_API_TOKEN")

	ctx := context.Background()
	p := New("test")()
	var schemaResp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)

	rawVal := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{"api_token": tftypes.String}},
		map[string]tftypes.Value{"api_token": tftypes.NewValue(tftypes.String, nil)},
	)
	req := provider.ConfigureRequest{
		Config: tfsdk.Config{Raw: rawVal, Schema: schemaResp.Schema},
	}
	var resp provider.ConfigureResponse
	p.Configure(ctx, req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Configure() expected error when token missing")
	}
	var found bool
	for _, d := range resp.Diagnostics {
		if d.Summary() == "Missing API token" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected diagnostic 'Missing API token', got: %s", resp.Diagnostics)
	}
}
