package acceptance

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSource_distributors(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "serverguard24_distributors" "all" {}`,
				Check: resource.TestCheckResourceAttrSet(
					"data.serverguard24_distributors.all", "distributors.#",
				),
			},
		},
	})
}

func TestAccDataSource_groups(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "serverguard24_groups" "all" {}`,
				Check: resource.TestCheckResourceAttrSet(
					"data.serverguard24_groups.all", "groups.#",
				),
			},
		},
	})
}

func TestAccDataSource_integrations(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "serverguard24_integrations" "all" {}`,
				Check: resource.TestCheckResourceAttrSet(
					"data.serverguard24_integrations.all", "integrations.#",
				),
			},
		},
	})
}

func TestAccDataSource_nodes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `data "serverguard24_nodes" "all" {}`,
				Check: resource.TestCheckResourceAttrSet(
					"data.serverguard24_nodes.all", "nodes.#",
				),
			},
		},
	})
}
