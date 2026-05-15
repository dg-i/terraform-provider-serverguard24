package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// sweepPrefix is the name prefix used by every acceptance test fixture. The
// sweepers only delete resources whose name starts with this prefix so they
// can never touch hand-crafted records that share the test account.
const sweepPrefix = "acc-"

// TestMain wires up sweepers so leftover resources from previous failed runs
// are cleaned up before tests start.
//
// Run sweepers explicitly with: `go test ./internal/acceptance/... -v -sweep=all`
func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func init() {
	// Note: there is no bulk-list API for checks (service_show takes an id), so
	// no dedicated check sweeper exists. Deleting the parent host on the
	// Serverguard24 side removes its checks transitively.
	resource.AddTestSweepers("serverguard24_host", &resource.Sweeper{
		Name: "serverguard24_host",
		F:    sweepHosts,
	})
	resource.AddTestSweepers("serverguard24_contact", &resource.Sweeper{
		Name: "serverguard24_contact",
		F:    sweepContacts,
	})
}

// sweeperClient is the client used by sweepers. Returns an error rather than
// panicking when SG24_API_TOKEN is missing so `-sweep=` runs without
// credentials simply skip every sweeper instead of crashing.
func sweeperClient() (*client.Client, error) {
	token := os.Getenv("SG24_API_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("SG24_API_TOKEN not set; cannot run sweepers")
	}
	return client.New(token, "sweeper"), nil
}

func sweepHosts(_ string) error {
	c, err := sweeperClient()
	if err != nil {
		return err
	}
	resp, err := c.Do(context.Background(), client.Request{"command": "get_hosts"})
	if err != nil {
		return fmt.Errorf("listing hosts: %w", err)
	}
	var hosts []struct {
		ID    json.Number `json:"id"`
		Alias string      `json:"alias"`
	}
	if err := json.Unmarshal(resp.Data, &hosts); err != nil {
		return fmt.Errorf("parsing host list: %w", err)
	}
	for _, h := range hosts {
		if !strings.HasPrefix(h.Alias, sweepPrefix) {
			continue
		}
		if _, err := c.Do(context.Background(), client.Request{"command": "host_delete", "id": h.ID.String()}); err != nil {
			log.Printf("[WARN] sweep: host_delete(%s): %v", h.ID, err)
		}
	}
	return nil
}

func sweepContacts(_ string) error {
	c, err := sweeperClient()
	if err != nil {
		return err
	}
	resp, err := c.Do(context.Background(), client.Request{"command": "get_contacts"})
	if err != nil {
		return fmt.Errorf("listing contacts: %w", err)
	}
	var contacts []struct {
		ID   json.Number `json:"id"`
		Name string      `json:"name"`
	}
	if err := json.Unmarshal(resp.Data, &contacts); err != nil {
		return fmt.Errorf("parsing contact list: %w", err)
	}
	for _, ct := range contacts {
		if !strings.HasPrefix(ct.Name, sweepPrefix) {
			continue
		}
		if _, err := c.Do(context.Background(), client.Request{"command": "contact_delete", "id": ct.ID.String()}); err != nil {
			log.Printf("[WARN] sweep: contact_delete(%s): %v", ct.ID, err)
		}
	}
	return nil
}
