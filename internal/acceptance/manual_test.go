package acceptance

// Manual create/destroy mode for interactive resource inspection.
//
// Normal acceptance tests always create AND destroy resources in the same run,
// making it impossible to inspect them in the Serverguard24 UI.  These two
// tests break the lifecycle into two separate commands:
//
//   make testacc-create   → creates resources, writes IDs to acc-manual-state.json
//   make testacc-destroy  → reads IDs from acc-manual-state.json and deletes them
//
// The tests bypass the Terraform testing framework entirely and call the
// Serverguard24 API directly, so no Terraform state is involved.

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/dg-i/terraform-provider-serverguard24/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// manualStateFilePath returns the path to the JSON file used to persist
// resource IDs between create and destroy runs.
// Override the default with the SG24_TEST_STATEFILE environment variable.
func manualStateFilePath() string {
	if f := os.Getenv("SG24_TEST_STATEFILE"); f != "" {
		return f
	}
	return "acc-manual-state.json"
}

type manualState struct {
	Suffix        string `json:"suffix"`
	HostID        string `json:"host_id,omitempty"`
	ContactID     string `json:"contact_id,omitempty"`
	CheckID       string `json:"check_id,omitempty"`
	DistributorID string `json:"distributor_id,omitempty"` // linked to the check, if any
	IntegrationID string `json:"integration_id,omitempty"` // linked to the check, if any
	DowntimeID    string `json:"downtime_id,omitempty"`    // NOTE: cannot be deleted via API
}

func (s *manualState) save(t *testing.T) {
	t.Helper()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	stateFile := manualStateFilePath()
	if err := os.WriteFile(stateFile, data, 0o644); err != nil {
		t.Fatalf("write state to %s: %v", stateFile, err)
	}
}

func newManualClient(t *testing.T) *client.Client {
	t.Helper()
	token := os.Getenv("SG24_API_TOKEN")
	if token == "" {
		t.Fatal("SG24_API_TOKEN must be set")
	}
	return client.New(token, "manual-test")
}

// firstIDOrEmpty calls the given API command, parses the response as a list of
// objects with an "id" field, and returns the first ID as a string.
// Returns "" (without failing) when the list is empty or access is denied,
// so callers can treat these resources as optional.
func firstIDOrEmpty(t *testing.T, c *client.Client, command string) string {
	t.Helper()
	resp, err := c.Do(t.Context(), client.Request{"command": command})
	if err != nil {
		if client.IsPermissionError(err) {
			return ""
		}
		t.Logf("warning: %s: %v", command, err)
		return ""
	}
	var items []struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		t.Logf("warning: parse %s response: %v", command, err)
		return ""
	}
	if len(items) == 0 {
		return ""
	}
	return items[0].ID.String()
}

// TestAccManual_Create creates a representative set of Serverguard24 resources
// via the API and saves their IDs to a local JSON file so they can be inspected
// in the Serverguard24 UI.
//
// Resources created:
//   - host + contact + ping check  (always)
//   - maintenance window (downtime) linked to the check  (always)
//   - check→distributor link       (when the account has distributors)
//   - check→integration link       (when the account has integrations)
//
// State is saved after each successful operation, so partial results can be
// cleaned up with TestAccManual_Destroy even if the test fails halfway.
//
// Run with:  make testacc-create
// Clean up:  make testacc-destroy
func TestAccManual_Create(t *testing.T) {
	if os.Getenv("SG24_TEST_MODE") != "create" {
		t.Skip("set SG24_TEST_MODE=create to run (use 'make testacc-create')")
	}
	testAccPreCheck(t)

	c := newManualClient(t)
	suffix := "acc-manual-" + acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	state := manualState{Suffix: suffix}

	// ── host ──────────────────────────────────────────────────────────────────
	hostResp, err := c.Do(t.Context(), client.Request{
		"command": "host_add",
		"name":    suffix + "-host",
		"address": "8.8.8.8",
	})
	if err != nil {
		t.Fatalf("host_add: %v", err)
	}
	var hostData struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal(hostResp.Data, &hostData); err != nil {
		t.Fatalf("parse host response: %v", err)
	}
	state.HostID = hostData.ID.String()
	state.save(t)
	t.Logf("host created    id=%-8s  name=%s", state.HostID, suffix+"-host")

	// ── contact ───────────────────────────────────────────────────────────────
	contactResp, err := c.Do(t.Context(), client.Request{
		"command":  "contact_add",
		"name":     suffix + "-contact",
		"language": "en",
		"email":    "acc-manual@example.com",
	})
	if err != nil {
		t.Fatalf("contact_add: %v", err)
	}
	var contactData struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal(contactResp.Data, &contactData); err != nil {
		t.Fatalf("parse contact response: %v", err)
	}
	state.ContactID = contactData.ID.String()
	state.save(t)
	t.Logf("contact created id=%-8s  name=%s", state.ContactID, suffix+"-contact")

	// ── ping check ────────────────────────────────────────────────────────────
	hostIDInt, _ := strconv.ParseInt(state.HostID, 10, 64)
	checkResp, err := c.Do(t.Context(), client.Request{
		"command":          "service_add",
		"host_id":          hostIDInt,
		"type":             40, // ping
		"shortname":        suffix + "-ping",
		"response_warning": 100,
		"response_critial": 500, // intentional API typo (missing 'c')
		"timeout":          10,
	})
	if err != nil {
		t.Fatalf("service_add (ping): %v", err)
	}
	var checkData struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal(checkResp.Data, &checkData); err != nil {
		t.Fatalf("parse check response: %v", err)
	}
	state.CheckID = checkData.ID.String()
	state.save(t)
	t.Logf("check created   id=%-8s  name=%s", state.CheckID, suffix+"-ping")

	checkIDInt, _ := strconv.ParseInt(state.CheckID, 10, 64)

	// ── downtime (maintenance window) ─────────────────────────────────────────
	// Near-future times: start at the next full hour, end two hours later.
	// Using near-future rather than far-future (2099) lets Serverguard24 auto-expire
	// and clean up the record, working around the missing downtime_delete API.
	startTime := time.Now().UTC().Add(1 * time.Hour).Truncate(time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	downtimeResp, err := c.Do(t.Context(), client.Request{
		"command":     "downtime_add",
		"hostname":    suffix + "-host",
		"starttime":   startTime.Unix(),
		"endtime":     endTime.Unix(),
		"description": suffix + "-downtime",
		"type":        1, // maintenance
	})
	if err != nil {
		t.Fatalf("downtime_add: %v", err)
	}
	var downtimeData struct {
		ID json.Number `json:"dt_id"` // API returns "dt_id", not "id"
	}
	if err := json.Unmarshal(downtimeResp.Data, &downtimeData); err != nil {
		t.Fatalf("parse downtime response: %v", err)
	}
	state.DowntimeID = downtimeData.ID.String()
	state.save(t)
	t.Logf("downtime created id=%-7s  desc=%s", state.DowntimeID, suffix+"-downtime")

	// Link the check to the downtime so it appears in the UI.
	if _, err := c.Do(t.Context(), client.Request{
		"command":  "downtime_link",
		"id":       state.DowntimeID,
		"services": []int64{checkIDInt},
	}); err != nil {
		t.Logf("downtime_link (services): %v (downtime created but check not linked)", err)
	} else {
		t.Logf("check linked to downtime")
	}

	// ── distributor link (optional) ───────────────────────────────────────────
	if distID := firstIDOrEmpty(t, c, "get_distributors"); distID != "" {
		distIDInt, _ := strconv.ParseInt(distID, 10, 64)
		if _, err := c.Do(t.Context(), client.Request{
			"command":        "service_link_distributor",
			"id":             checkIDInt,
			"distributor_id": distIDInt,
		}); err != nil {
			t.Logf("service_link_distributor: %v (skipping distributor link)", err)
		} else {
			state.DistributorID = distID
			state.save(t)
			t.Logf("distributor linked  id=%s", distID)
		}
	} else {
		t.Log("no distributors available — skipping distributor link")
	}

	// ── integration link (optional) ───────────────────────────────────────────
	if intID := firstIDOrEmpty(t, c, "get_integrations"); intID != "" {
		intIDInt, _ := strconv.ParseInt(intID, 10, 64)
		if _, err := c.Do(t.Context(), client.Request{
			"command":        "service_link_integration",
			"id":             checkIDInt,
			"integration_id": intIDInt,
		}); err != nil {
			t.Logf("service_link_integration: %v (skipping integration link)", err)
		} else {
			state.IntegrationID = intID
			state.save(t)
			t.Logf("integration linked  id=%s", intID)
		}
	} else {
		t.Log("no integrations available — skipping integration link")
	}

	// ── done ──────────────────────────────────────────────────────────────────
	t.Logf("state file: %s", manualStateFilePath())
	t.Log("Inspect the resources in the Serverguard24 UI, then run 'make testacc-destroy' to clean up.")
}

// TestAccManual_Destroy deletes the resources that were created by
// TestAccManual_Create.  It reads resource IDs from the JSON state file and
// cleans up each resource, then removes the state file.
//
// Deletion order: links → check → downtime (unlink only) → contact → host.
//
// KNOWN LIMITATION: The Serverguard24 API has no downtime_delete command.
// The downtime record is unlinked from the check but its entry persists in the
// Serverguard24 UI and must be removed manually.
//
// Run with:  make testacc-destroy
func TestAccManual_Destroy(t *testing.T) {
	if os.Getenv("SG24_TEST_MODE") != "destroy" {
		t.Skip("set SG24_TEST_MODE=destroy to run (use 'make testacc-destroy')")
	}
	testAccPreCheck(t)

	stateFile := manualStateFilePath()
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state from %s: %v — run 'make testacc-create' first", stateFile, err)
	}
	var state manualState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("parse state: %v", err)
	}

	c := newManualClient(t)
	anyFailed := false

	checkIDInt, _ := strconv.ParseInt(state.CheckID, 10, 64)

	// ── unlink distributor ────────────────────────────────────────────────────
	if state.DistributorID != "" {
		distIDInt, _ := strconv.ParseInt(state.DistributorID, 10, 64)
		if _, err := c.Do(t.Context(), client.Request{
			"command":        "service_unlink_distributor",
			"id":             checkIDInt,
			"distributor_id": distIDInt,
		}); err != nil {
			t.Errorf("service_unlink_distributor %s: %v", state.DistributorID, err)
			anyFailed = true
		} else {
			t.Logf("distributor unlinked  id=%s", state.DistributorID)
		}
	}

	// ── unlink integration ────────────────────────────────────────────────────
	if state.IntegrationID != "" {
		intIDInt, _ := strconv.ParseInt(state.IntegrationID, 10, 64)
		if _, err := c.Do(t.Context(), client.Request{
			"command":        "service_unlink_integration",
			"id":             checkIDInt,
			"integration_id": intIDInt,
		}); err != nil {
			t.Errorf("service_unlink_integration %s: %v", state.IntegrationID, err)
			anyFailed = true
		} else {
			t.Logf("integration unlinked  id=%s", state.IntegrationID)
		}
	}

	// ── delete check ──────────────────────────────────────────────────────────
	if state.CheckID != "" {
		if _, err := c.Do(t.Context(), client.Request{"command": "service_delete", "id": state.CheckID}); err != nil {
			t.Errorf("service_delete %s: %v", state.CheckID, err)
			anyFailed = true
		} else {
			t.Logf("check deleted   id=%s", state.CheckID)
		}
	}

	// ── downtime: unlink only (no downtime_delete in API) ─────────────────────
	if state.DowntimeID != "" && state.CheckID != "" {
		if _, err := c.Do(t.Context(), client.Request{
			"command":  "downtime_unlink",
			"id":       state.DowntimeID,
			"services": []int64{checkIDInt},
		}); err != nil {
			t.Logf("downtime_unlink (services): %v (link may already be gone)", err)
		} else {
			t.Logf("downtime unlinked  id=%s", state.DowntimeID)
		}
		t.Logf("WARNING: downtime id=%s cannot be deleted via the API — remove it manually in the Serverguard24 UI", state.DowntimeID)
	}

	// ── delete contact ────────────────────────────────────────────────────────
	if state.ContactID != "" {
		if _, err := c.Do(t.Context(), client.Request{"command": "contact_delete", "id": state.ContactID}); err != nil {
			t.Errorf("contact_delete %s: %v", state.ContactID, err)
			anyFailed = true
		} else {
			t.Logf("contact deleted id=%s", state.ContactID)
		}
	}

	// ── delete host ───────────────────────────────────────────────────────────
	if state.HostID != "" {
		if _, err := c.Do(t.Context(), client.Request{"command": "host_delete", "id": state.HostID}); err != nil {
			t.Errorf("host_delete %s: %v", state.HostID, err)
			anyFailed = true
		} else {
			t.Logf("host deleted    id=%s", state.HostID)
		}
	}

	if !anyFailed {
		if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
			t.Logf("warning: could not remove state file %s: %v", stateFile, err)
		} else {
			t.Logf("state file %s removed", stateFile)
		}
	} else {
		t.Logf("some deletions failed — state file %s kept for retry", stateFile)
	}
}
