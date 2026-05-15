package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// apiClient returns a real Serverguard24 client for use in CheckDestroy and
// sweeper functions. The token comes from SG24_API_TOKEN (testAccPreCheck
// already enforces presence).
func apiClient() *client.Client {
	return client.New(os.Getenv("SG24_API_TOKEN"), "test")
}

// checkResourceGone returns nil if no resource with type tfType remains in
// state. lookup is called once per matching state entry with its id and must
// return nil if the resource is confirmed absent from the API, or an error if
// it is still present (or the lookup itself failed unexpectedly).
//
// "Not found" is matched via client.IsNotFound, which is the same semantics
// resources use to recognise a deleted record.
func checkResourceGone(tfType string, lookup func(id string) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != tfType {
				continue
			}
			if rs.Primary.ID == "" {
				continue
			}
			if err := lookup(rs.Primary.ID); err != nil {
				return err
			}
		}
		return nil
	}
}

// idStillPresent unmarshals the API data as a list of objects with an "id"
// field and reports whether wantID is in the list.
func idStillPresent(data json.RawMessage, wantID string) bool {
	var items []struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return false
	}
	for _, it := range items {
		if it.ID.String() == wantID {
			return true
		}
	}
	return false
}

func testAccCheckHostDestroy(s *terraform.State) error {
	return checkResourceGone("serverguard24_host", func(id string) error {
		c := apiClient()
		resp, err := c.Do(context.Background(), client.Request{"command": "get_hosts", "id": id})
		if err != nil {
			if client.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("get_hosts(%s) failed: %w", id, err)
		}
		if idStillPresent(resp.Data, id) {
			return fmt.Errorf("host %s still exists in the API after destroy", id)
		}
		return nil
	})(s)
}

func testAccCheckContactDestroy(s *terraform.State) error {
	return checkResourceGone("serverguard24_contact", func(id string) error {
		c := apiClient()
		resp, err := c.Do(context.Background(), client.Request{"command": "get_contacts", "id": id})
		if err != nil {
			if client.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("get_contacts(%s) failed: %w", id, err)
		}
		if idStillPresent(resp.Data, id) {
			return fmt.Errorf("contact %s still exists in the API after destroy", id)
		}
		return nil
	})(s)
}

func testAccCheckCheckDestroy(s *terraform.State) error {
	return checkResourceGone("serverguard24_check", func(id string) error {
		c := apiClient()
		_, err := c.Do(context.Background(), client.Request{"command": "service_show", "id": id})
		if err != nil {
			if client.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("service_show(%s) failed: %w", id, err)
		}
		return fmt.Errorf("check %s still exists in the API after destroy", id)
	})(s)
}

// testAccCheckDowntimeDestroy is intentionally permissive: the Serverguard24
// API has no downtime_delete command, so destroyed downtimes remain in the
// account until they expire naturally. We only verify that the host/check
// that the downtime referenced was cleaned up by relying on the corresponding
// host/check destroy verifiers; the downtime itself is expected to persist.
func testAccCheckDowntimeDestroy(_ *terraform.State) error {
	return nil
}
