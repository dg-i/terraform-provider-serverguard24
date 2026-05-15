package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHost_basic(t *testing.T) {
	rName := "acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckHostDestroy,
		Steps: []resource.TestStep{
			// Create and read
			{
				Config: providerConfig + fmt.Sprintf(`
resource "serverguard24_host" "test" {
  name    = %q
  address = "8.8.8.8"
}`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("serverguard24_host.test", "name", rName),
					resource.TestCheckResourceAttr("serverguard24_host.test", "address", "8.8.8.8"),
					resource.TestCheckResourceAttrSet("serverguard24_host.test", "id"),
				),
			},
			// Import
			{
				ResourceName:      "serverguard24_host.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name and address
			{
				Config: providerConfig + fmt.Sprintf(`
resource "serverguard24_host" "test" {
  name    = %q
  address = "8.8.4.4"
}`, rName+"-upd"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("serverguard24_host.test", "name", rName+"-upd"),
					resource.TestCheckResourceAttr("serverguard24_host.test", "address", "8.8.4.4"),
				),
			},
		},
	})
}
