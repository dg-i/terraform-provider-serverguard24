package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const defaultAPIEndpoint = "https://www.serverguard24.de/de/api/json.html"

// Client is the Serverguard24 API client.
type Client struct {
	token      string
	endpoint   string
	userAgent  string
	httpClient *retryablehttp.Client

	hostsCache    *listCache
	contactsCache *listCache
}

// New creates a new API client using the default Serverguard24 API endpoint.
func New(token, version string) *Client {
	return NewWithEndpoint(token, version, defaultAPIEndpoint)
}

// NewWithEndpoint creates a new API client with a custom endpoint (e.g. for testing).
func NewWithEndpoint(token, version, endpoint string) *Client {
	rc := retryablehttp.NewClient()
	rc.Logger = nil
	rc.RetryMax = 3
	rc.RetryWaitMin = 500 * time.Millisecond
	rc.RetryWaitMax = 5 * time.Second
	rc.HTTPClient.Timeout = 30 * time.Second
	// Pass the final response through so our own status-code check produces a
	// usable error (e.g. "HTTP status 500") instead of retryablehttp's generic
	// "giving up after N attempts" wrapper.
	rc.ErrorHandler = retryablehttp.PassthroughErrorHandler

	return &Client{
		token:         token,
		endpoint:      endpoint,
		userAgent:     "terraform-provider-serverguard24/" + version,
		httpClient:    rc,
		hostsCache:    newListCache(30 * time.Second),
		contactsCache: newListCache(30 * time.Second),
	}
}

// Request is a single command in the batch request body.
type Request map[string]any

// Response is a single response element returned by the API.
// The API may return "code" as either a JSON string ("100") or a number (100).
// Error messages may be returned as either "message" or "errmessage".
type Response struct {
	Code       json.RawMessage `json:"code"`
	Message    string          `json:"message"`
	ErrMessage string          `json:"errmessage"`
	Data       json.RawMessage `json:"data"`
}

// codeString returns the code value as a plain string regardless of JSON type.
func (r *Response) codeString() string {
	return strings.Trim(string(r.Code), `"`)
}

func (r *Response) errorMessage() string {
	if r.Message != "" {
		return r.Message
	}
	return r.ErrMessage
}

// Do sends a single command to the API and returns the response.
// The token is sent only in the outer body for authentication.
// The supplied ctx is honored for cancellation and propagated to retries.
func (c *Client) Do(ctx context.Context, cmd Request) (*Response, error) {
	body := struct {
		Token    string    `json:"token"`
		Requests []Request `json:"requests"`
	}{
		Token:    c.token,
		Requests: []Request{cmd},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var responses []Response
	if err := json.Unmarshal(raw, &responses); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	r := &responses[0]
	if r.codeString() != "100" {
		return nil, &APIError{Code: r.codeString(), Message: r.errorMessage()}
	}

	return r, nil
}

// APIError is returned when the Serverguard24 API responds with a non-100 code.
// Callers should use errors.As to distinguish API errors from transport errors,
// and the Is* helpers (IsNotFound, IsPermissionError, IsAlreadyExists) for the
// common error-code classifications.
type APIError struct {
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %s: %s", e.Code, e.Message)
}

// codeIs reports whether err wraps an APIError with one of the given codes.
func codeIs(err error, codes ...string) bool {
	if err == nil {
		return false
	}
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	return slices.Contains(codes, ae.Code)
}

// IsNotFound returns true when the error looks like a missing resource.
//
// Two codes are mapped to "not found":
//   - 210 ("invalid parameter") — returned by list endpoints (get_hosts,
//     get_contacts) when the supplied id is unknown; also returned for genuinely
//     malformed requests, so this can over-match.
//   - 300 (generic error, empty message) — returned by service_show when the
//     check id has been deleted. Undocumented but observed in practice.
//
// Resources treat such errors as "gone" and remove the resource from state,
// which is the safest behaviour given the API's lack of a dedicated not-found
// code.
func IsNotFound(err error) bool {
	return codeIs(err, "210", "300")
}

// IsPermissionError returns true for auth (220) or insufficient rights (270) errors.
// Data sources use this to degrade gracefully instead of blocking the plan.
func IsPermissionError(err error) bool {
	return codeIs(err, "220", "270")
}

// IsAlreadyExists returns true when the API reports the resource already exists (310).
func IsAlreadyExists(err error) bool {
	return codeIs(err, "310")
}

// listCache is a single-slot cache for endpoints like get_hosts / get_contacts
// whose id filter the API ignores. The first caller within the TTL pays the
// network round-trip; concurrent callers block on the mutex and reuse the
// result. CRUD operations explicitly invalidate the cache so a follow-up Read
// in the same Terraform run sees its own writes.
type listCache struct {
	ttl time.Duration

	mu       sync.Mutex
	fetched  time.Time
	response *Response
	err      error
}

func newListCache(ttl time.Duration) *listCache {
	return &listCache{ttl: ttl}
}

func (lc *listCache) get(fetch func() (*Response, error)) (*Response, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if lc.response != nil && time.Since(lc.fetched) < lc.ttl {
		return lc.response, lc.err
	}
	lc.response, lc.err = fetch()
	lc.fetched = time.Now()
	return lc.response, lc.err
}

func (lc *listCache) invalidate() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.response = nil
	lc.err = nil
	lc.fetched = time.Time{}
}

// ListHosts returns the (possibly cached) get_hosts response. Callers must
// treat the returned *Response as read-only — it is shared across concurrent
// callers within the TTL window.
func (c *Client) ListHosts(ctx context.Context) (*Response, error) {
	return c.hostsCache.get(func() (*Response, error) {
		return c.Do(ctx, Request{"command": "get_hosts"})
	})
}

// InvalidateHosts drops the cached get_hosts response. CRUD operations on
// hosts must call this so a follow-up Read sees their changes.
func (c *Client) InvalidateHosts() {
	c.hostsCache.invalidate()
}

// ListContacts returns the (possibly cached) get_contacts response. See
// ListHosts for caveats.
func (c *Client) ListContacts(ctx context.Context) (*Response, error) {
	return c.contactsCache.get(func() (*Response, error) {
		return c.Do(ctx, Request{"command": "get_contacts"})
	})
}

// InvalidateContacts drops the cached get_contacts response.
func (c *Client) InvalidateContacts() {
	c.contactsCache.invalidate()
}
