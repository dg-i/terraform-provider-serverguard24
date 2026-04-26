// Package acceptance contains acceptance tests that run against the real Serverguard24 API.
// Set TF_ACC=1 and SG24_API_TOKEN to run these tests.
package acceptance

import (
	"os"
	"testing"

	"github.com/dg-i/terraform-provider-serverguard24/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories instantiates the provider under test using
// the same code path as the real binary.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"serverguard24": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// testAccPreCheck verifies required environment variables are set before any
// acceptance test runs.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("SG24_API_TOKEN") == "" {
		t.Fatal("SG24_API_TOKEN must be set for acceptance tests")
	}
}

// providerConfig is the minimal provider block that reads the API token from
// the SG24_API_TOKEN environment variable.
const providerConfig = `
provider "serverguard24" {}
`
