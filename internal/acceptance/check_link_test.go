package acceptance

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// sharedCheckContactConfig creates a host, contact, and ping check — used by
// contact-link tests.
func sharedCheckContactConfig(rName string) string {
	return providerConfig + fmt.Sprintf(`
resource "serverguard24_host" "test" {
  name    = %q
  address = "8.8.8.8"
}

resource "serverguard24_contact" "test" {
  name     = %q
  language = "en"
  email    = "acc-link@example.com"
}

resource "serverguard24_check" "test" {
  host_id   = serverguard24_host.test.id
  type      = "ping"
  shortname = %q
  active    = false
  params {
    response_warning  = 100
    response_critical = 500
    timeout           = 10
  }
}
`, rName+"-host", rName+"-contact", rName+"-check")
}

// sharedPingCheckConfig creates a host and ping check without a contact.
// Used by distributor- and integration-link tests.
func sharedPingCheckConfig(rName string) string {
	return providerConfig + fmt.Sprintf(`
resource "serverguard24_host" "test" {
  name    = %q
  address = "8.8.8.8"
}

resource "serverguard24_check" "test" {
  host_id   = serverguard24_host.test.id
  type      = "ping"
  shortname = %q
  active    = false
  params {
    response_warning  = 100
    response_critical = 500
    timeout           = 10
  }
}
`, rName+"-host", rName+"-check")
}

// findFirstID calls the given API command, parses the response as a list of
// objects with an "id" field, and returns the first ID as a string.
// If the list is empty or the account has no permission, t.Skip() is called.
func findFirstID(t *testing.T, command string, skipMsg string) string {
	t.Helper()
	c := client.New(os.Getenv("SG24_API_TOKEN"), "test")
	resp, err := c.Do(client.Request{"command": command})
	if err != nil {
		if client.IsPermissionError(err) {
			t.Skipf("%s: no permission — %v", skipMsg, err)
		}
		t.Fatalf("%s: API error — %v", command, err)
	}
	var items []struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		t.Fatalf("%s: parse response: %v", command, err)
	}
	if len(items) == 0 {
		t.Skipf("%s: no items available — %s", command, skipMsg)
	}
	return items[0].ID.String()
}

// ── contact link ──────────────────────────────────────────────────────────────

// TestAccCheckContactLink_basic creates a check-contact link and verifies the
// attribute values.  An import step exercises ImportState using the composite
// "check_id/contact_id" ID that the resource stores.
func TestAccCheckContactLink_basic(t *testing.T) {
	rName := "acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify attribute values.
			{
				Config: sharedCheckContactConfig(rName) + `
resource "serverguard24_check_contact_link" "test" {
  check_id   = serverguard24_check.test.id
  contact_id = serverguard24_contact.test.id
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("serverguard24_check_contact_link.test", "id"),
					// Verify the stored IDs actually match the referenced resources.
					resource.TestCheckResourceAttrPair(
						"serverguard24_check_contact_link.test", "check_id",
						"serverguard24_check.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"serverguard24_check_contact_link.test", "contact_id",
						"serverguard24_contact.test", "id",
					),
				),
			},
			// Import by composite ID (check_id/contact_id) and verify state matches.
			{
				ResourceName:      "serverguard24_check_contact_link.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccCheckContactLink_recreation verifies that changing the contact_id
// destroys the old link and creates a new one (both attributes carry
// RequiresReplace).
func TestAccCheckContactLink_recreation(t *testing.T) {
	rName := "acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	// Config with contact A linked.
	configA := sharedCheckContactConfig(rName) + fmt.Sprintf(`
resource "serverguard24_contact" "second" {
  name     = %q
  language = "en"
  email    = "acc-link-b@example.com"
}

resource "serverguard24_check_contact_link" "test" {
  check_id   = serverguard24_check.test.id
  contact_id = serverguard24_contact.test.id
}`, rName+"-contact-b")

	// Config with contact B linked (triggers destroy + recreate of the link).
	configB := sharedCheckContactConfig(rName) + fmt.Sprintf(`
resource "serverguard24_contact" "second" {
  name     = %q
  language = "en"
  email    = "acc-link-b@example.com"
}

resource "serverguard24_check_contact_link" "test" {
  check_id   = serverguard24_check.test.id
  contact_id = serverguard24_contact.second.id
}`, rName+"-contact-b")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Link to contact A.
			{
				Config: configA,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("serverguard24_check_contact_link.test", "id"),
					resource.TestCheckResourceAttrPair(
						"serverguard24_check_contact_link.test", "contact_id",
						"serverguard24_contact.test", "id",
					),
				),
			},
			// Swap to contact B — link must be destroyed and recreated.
			{
				Config: configB,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("serverguard24_check_contact_link.test", "id"),
					resource.TestCheckResourceAttrPair(
						"serverguard24_check_contact_link.test", "contact_id",
						"serverguard24_contact.second", "id",
					),
				),
			},
		},
	})
}

// ── distributor link ──────────────────────────────────────────────────────────

// TestAccCheckDistributorLink_basic links a check to the first available
// distributor.  It also exercises the serverguard24_distributors data source,
// verifying that item-level attributes (id, name) are populated.
// The test is skipped automatically when no distributors are configured on the
// account.
func TestAccCheckDistributorLink_basic(t *testing.T) {
	rName := "acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	distributorID := findFirstID(t, "get_distributors",
		"no distributors configured — skipping distributor link test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create the link and verify both the data source and the link resource.
			{
				Config: sharedPingCheckConfig(rName) + fmt.Sprintf(`
data "serverguard24_distributors" "all" {}

resource "serverguard24_check_distributor_link" "test" {
  check_id       = serverguard24_check.test.id
  distributor_id = %q
}`, distributorID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Data source: at least one distributor with populated fields.
					resource.TestCheckResourceAttrSet("data.serverguard24_distributors.all", "distributors.#"),
					resource.TestCheckResourceAttrSet("data.serverguard24_distributors.all", "distributors.0.id"),
					resource.TestCheckResourceAttrSet("data.serverguard24_distributors.all", "distributors.0.name"),
					// Link resource.
					resource.TestCheckResourceAttrSet("serverguard24_check_distributor_link.test", "id"),
					resource.TestCheckResourceAttrPair(
						"serverguard24_check_distributor_link.test", "check_id",
						"serverguard24_check.test", "id",
					),
					resource.TestCheckResourceAttr(
						"serverguard24_check_distributor_link.test", "distributor_id", distributorID,
					),
				),
			},
			// Import by composite ID (check_id/distributor_id).
			{
				ResourceName:      "serverguard24_check_distributor_link.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// ── integration link ──────────────────────────────────────────────────────────

// TestAccCheckIntegrationLink_basic links a check to the first available
// integration.  It also exercises the serverguard24_integrations data source,
// verifying that item-level attributes (id, name) are populated.
// The test is skipped automatically when no integrations are configured on the
// account.
func TestAccCheckIntegrationLink_basic(t *testing.T) {
	rName := "acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	integrationID := findFirstID(t, "get_integrations",
		"no integrations configured — skipping integration link test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create the link and verify both the data source and the link resource.
			{
				Config: sharedPingCheckConfig(rName) + fmt.Sprintf(`
data "serverguard24_integrations" "all" {}

resource "serverguard24_check_integration_link" "test" {
  check_id       = serverguard24_check.test.id
  integration_id = %q
}`, integrationID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Data source: at least one integration with populated fields.
					resource.TestCheckResourceAttrSet("data.serverguard24_integrations.all", "integrations.#"),
					resource.TestCheckResourceAttrSet("data.serverguard24_integrations.all", "integrations.0.id"),
					resource.TestCheckResourceAttrSet("data.serverguard24_integrations.all", "integrations.0.name"),
					// Link resource.
					resource.TestCheckResourceAttrSet("serverguard24_check_integration_link.test", "id"),
					resource.TestCheckResourceAttrPair(
						"serverguard24_check_integration_link.test", "check_id",
						"serverguard24_check.test", "id",
					),
					resource.TestCheckResourceAttr(
						"serverguard24_check_integration_link.test", "integration_id", integrationID,
					),
				),
			},
			// Import by composite ID (check_id/integration_id).
			{
				ResourceName:      "serverguard24_check_integration_link.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
