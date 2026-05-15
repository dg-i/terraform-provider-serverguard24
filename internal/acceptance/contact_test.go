package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccContact_basic(t *testing.T) {
	rName := "acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckContactDestroy,
		Steps: []resource.TestStep{
			// Create and read
			{
				Config: providerConfig + fmt.Sprintf(`
resource "serverguard24_contact" "test" {
  name     = %q
  language = "en"
  email    = "acc-test@example.com"
}`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("serverguard24_contact.test", "name", rName),
					resource.TestCheckResourceAttr("serverguard24_contact.test", "language", "en"),
					resource.TestCheckResourceAttr("serverguard24_contact.test", "email", "acc-test@example.com"),
					resource.TestCheckResourceAttrSet("serverguard24_contact.test", "id"),
				),
			},
			// Import
			{
				ResourceName:      "serverguard24_contact.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name and language
			{
				Config: providerConfig + fmt.Sprintf(`
resource "serverguard24_contact" "test" {
  name     = %q
  language = "de"
  email    = "acc-test-updated@example.com"
}`, rName+"-upd"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("serverguard24_contact.test", "name", rName+"-upd"),
					resource.TestCheckResourceAttr("serverguard24_contact.test", "language", "de"),
					resource.TestCheckResourceAttr("serverguard24_contact.test", "email", "acc-test-updated@example.com"),
				),
			},
		},
	})
}
