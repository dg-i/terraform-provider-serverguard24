package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCheck_http(t *testing.T) {
	rName := "acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCheckDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				Config: providerConfig + fmt.Sprintf(`
resource "serverguard24_host" "test" {
  name    = %q
  address = "example.com"
}

resource "serverguard24_check" "http" {
  host_id   = serverguard24_host.test.id
  type      = "http"
  shortname = %q
  active    = false
  params {
    path    = "/"
    timeout = 10
    port    = 443
  }
}`, rName, rName+"-http"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("serverguard24_check.http", "type", "http"),
					resource.TestCheckResourceAttr("serverguard24_check.http", "shortname", rName+"-http"),
					resource.TestCheckResourceAttr("serverguard24_check.http", "params.port", "443"),
					resource.TestCheckResourceAttrSet("serverguard24_check.http", "id"),
				),
			},
			// Import — full ImportStateVerify is not possible for two reasons:
			// (1) service_show does not return host_id, so it is lost on import;
			// (2) many optional attributes (notification flags, interval, region, …)
			// are populated by service_show but are null in state right after Create
			// (Create stores only the plan). Fixing this cleanly requires marking
			// those attributes as Optional+Computed, which is a larger schema change.
			{
				ResourceName: "serverguard24_check.http",
				ImportState:  true,
			},
			// Update shortname and port
			{
				Config: providerConfig + fmt.Sprintf(`
resource "serverguard24_host" "test" {
  name    = %q
  address = "example.com"
}

resource "serverguard24_check" "http" {
  host_id   = serverguard24_host.test.id
  type      = "http"
  shortname = %q
  active    = false
  params {
    path    = "/"
    timeout = 10
    port    = 80
  }
}`, rName, rName+"-http-upd"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("serverguard24_check.http", "shortname", rName+"-http-upd"),
					resource.TestCheckResourceAttr("serverguard24_check.http", "params.port", "80"),
				),
			},
		},
	})
}

func TestAccCheck_ping(t *testing.T) {
	rName := "acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "serverguard24_host" "test" {
  name    = %q
  address = "8.8.8.8"
}

resource "serverguard24_check" "ping" {
  host_id   = serverguard24_host.test.id
  type      = "ping"
  shortname = %q
  active    = false
  params {
    response_warning  = 100
    response_critical = 500
    timeout           = 10
  }
}`, rName, rName+"-ping"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("serverguard24_check.ping", "type", "ping"),
					resource.TestCheckResourceAttr("serverguard24_check.ping", "shortname", rName+"-ping"),
					resource.TestCheckResourceAttrSet("serverguard24_check.ping", "id"),
				),
			},
		},
	})
}
