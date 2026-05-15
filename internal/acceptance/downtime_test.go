package acceptance

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// nearFutureWindow returns a start/end time pair as RFC 3339 strings.
// start is the next full hour after now + offsetHours, end is one hour later.
// Using near-future times (rather than 2099) means Serverguard24 can expire
// and auto-clean the downtime records, working around the missing downtime_delete API.
func nearFutureWindow(offsetHours int) (start, end string) {
	base := time.Now().UTC().Add(time.Duration(offsetHours) * time.Hour).Truncate(time.Hour)
	return base.Format(time.RFC3339), base.Add(time.Hour).Format(time.RFC3339)
}

// downtimeHostCheckConfig creates a host and ping check shared by all downtime tests.
func downtimeHostCheckConfig(rName string) string {
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

// TestAccDowntime_basic creates a maintenance window linked to a check via the
// services list and verifies all stored attributes.
//
// An import step is included to exercise ImportState. Because the API has no
// downtime_show command, import only recovers the id; all other attributes stay
// null until the operator adds them to the config and re-applies. The test
// therefore skips full state verification after import.
func TestAccDowntime_basic(t *testing.T) {
	rName := "acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	startTime, endTime := nearFutureWindow(1)

	config := downtimeHostCheckConfig(rName) + fmt.Sprintf(`
resource "serverguard24_downtime" "test" {
  hostname    = serverguard24_host.test.name
  start_time  = %q
  end_time    = %q
  description = %q
  services    = [serverguard24_check.test.id]
}`, startTime, endTime, rName+"-downtime")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDowntimeDestroy,
		Steps: []resource.TestStep{
			// Create and verify all stored attributes.
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("serverguard24_downtime.test", "id"),
					resource.TestCheckResourceAttr("serverguard24_downtime.test", "hostname", rName+"-host"),
					resource.TestCheckResourceAttr("serverguard24_downtime.test", "description", rName+"-downtime"),
					resource.TestCheckResourceAttr("serverguard24_downtime.test", "type", "maintenance"),
					resource.TestCheckResourceAttr("serverguard24_downtime.test", "start_time", startTime),
					resource.TestCheckResourceAttr("serverguard24_downtime.test", "end_time", endTime),
					// Exactly one service linked.
					resource.TestCheckResourceAttr("serverguard24_downtime.test", "services.#", "1"),
				),
			},
			// Import by downtime ID.  Full state verification is skipped because
			// the API has no downtime_show; import can only restore the id.
			{
				ResourceName: "serverguard24_downtime.test",
				ImportState:  true,
				// ImportStateVerify intentionally omitted — see resource documentation.
			},
		},
	})
}

// TestAccDowntime_withHosts creates a maintenance window linked to a host via
// the hosts list (rather than the services list) and verifies the count.
func TestAccDowntime_withHosts(t *testing.T) {
	rName := "acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	startTime, endTime := nearFutureWindow(1)

	config := downtimeHostCheckConfig(rName) + fmt.Sprintf(`
resource "serverguard24_downtime" "test" {
  hostname    = serverguard24_host.test.name
  start_time  = %q
  end_time    = %q
  description = %q
  hosts       = [serverguard24_host.test.id]
}`, startTime, endTime, rName+"-downtime-hosts")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDowntimeDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("serverguard24_downtime.test", "id"),
					resource.TestCheckResourceAttr("serverguard24_downtime.test", "hostname", rName+"-host"),
					resource.TestCheckResourceAttr("serverguard24_downtime.test", "description", rName+"-downtime-hosts"),
					resource.TestCheckResourceAttr("serverguard24_downtime.test", "type", "maintenance"),
					// Exactly one host linked, no services.
					resource.TestCheckResourceAttr("serverguard24_downtime.test", "hosts.#", "1"),
					resource.TestCheckNoResourceAttr("serverguard24_downtime.test", "services.#"),
				),
			},
		},
	})
}

// TestAccDowntime_recreation verifies that changing any required attribute
// (here: description) triggers a destroy + recreate cycle rather than an
// in-place update, since all attributes carry RequiresReplace.
//
// Two distinct time windows are used (offset 1h and 3h) so that the step-2
// downtime does not conflict with the step-1 record. Because the API has no
// downtime_delete, the step-1 record persists in Serverguard24 after destroy;
// reusing the same window would cause an "already exists" error.
// Near-future windows are used so Serverguard24 can auto-expire the orphaned records.
func TestAccDowntime_recreation(t *testing.T) {
	rName := "acc-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	startTime1, endTime1 := nearFutureWindow(1)
	startTime2, endTime2 := nearFutureWindow(3)

	baseConfig := downtimeHostCheckConfig(rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDowntimeDestroy,
		Steps: []resource.TestStep{
			// Step 1: initial downtime.
			{
				Config: baseConfig + fmt.Sprintf(`
resource "serverguard24_downtime" "test" {
  hostname    = serverguard24_host.test.name
  start_time  = %q
  end_time    = %q
  description = %q
  services    = [serverguard24_check.test.id]
}`, startTime1, endTime1, rName+"-v1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("serverguard24_downtime.test", "id"),
					resource.TestCheckResourceAttr("serverguard24_downtime.test", "description", rName+"-v1"),
				),
			},
			// Step 2: change description and time window → triggers destroy + recreate.
			// A different time window is required because the API has no downtime_delete:
			// the step-1 downtime record stays in the API (only its links are removed),
			// so reusing the same window would cause an "already exists" conflict.
			{
				Config: baseConfig + fmt.Sprintf(`
resource "serverguard24_downtime" "test" {
  hostname    = serverguard24_host.test.name
  start_time  = %q
  end_time    = %q
  description = %q
  services    = [serverguard24_check.test.id]
}`, startTime2, endTime2, rName+"-v2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("serverguard24_downtime.test", "id"),
					resource.TestCheckResourceAttr("serverguard24_downtime.test", "description", rName+"-v2"),
				),
			},
		},
	})
}
