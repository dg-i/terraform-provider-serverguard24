package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultAPIEndpoint = "https://www.serverguard24.de/de/api/json.html"

// Client is the Serverguard24 API client.
type Client struct {
	token      string
	endpoint   string
	userAgent  string
	httpClient *http.Client
}

// New creates a new API client using the default Serverguard24 API endpoint.
func New(token, version string) *Client {
	return NewWithEndpoint(token, version, defaultAPIEndpoint)
}

// NewWithEndpoint creates a new API client with a custom endpoint (e.g. for testing).
func NewWithEndpoint(token, version, endpoint string) *Client {
	return &Client{
		token:      token,
		endpoint:   endpoint,
		userAgent:  "terraform-provider-serverguard24/" + version,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Request is a single command in the batch request body.
type Request map[string]interface{}

// Response is a single response element returned by the API.
// The API may return "code" as either a JSON string ("100") or a number (100).
// The message field is named "message" in actual API responses (not "errmessage"
// as stated in the API documentation).
type Response struct {
	Code       json.RawMessage `json:"code"`
	ErrMessage string          `json:"message"`
	Data       json.RawMessage `json:"data"`
}

// codeString returns the code value as a plain string regardless of JSON type.
func (r *Response) codeString() string {
	return strings.Trim(string(r.Code), `"`)
}

// Do sends a single command to the API and returns the response.
// The API requires the token both in the outer body and inside each request.
func (c *Client) Do(cmd Request) (*Response, error) {
	// Copy cmd to avoid mutating the caller's map
	req := make(Request, len(cmd)+1)
	for k, v := range cmd {
		req[k] = v
	}
	req["token"] = c.token

	// Use a struct (not a map) to guarantee token appears before requests in the
	// serialized JSON. The API server appears to require this field ordering.
	body := struct {
		Token    string    `json:"token"`
		Requests []Request `json:"requests"`
	}{
		Token:    c.token,
		Requests: []Request{req},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

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
		return nil, fmt.Errorf("API error %s: %s", r.codeString(), r.ErrMessage)
	}

	return r, nil
}

// IsNotFound returns true when the error looks like a missing resource.
//
// Limitation: the API uses code 210 ("invalid parameter") both for unknown IDs
// and for genuinely malformed requests (wrong field type, bad command name, etc.).
// There is no distinct "not found" code, so this function may return true for
// parameter errors as well. Resources treat such errors as "gone" and remove
// the resource from state, which is the safest behaviour given the API design.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "API error 210")
}

// IsPermissionError returns true for auth (220) or insufficient rights (270) errors.
// Data sources use this to degrade gracefully instead of blocking the plan.
func IsPermissionError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "API error 220") ||
		strings.Contains(err.Error(), "API error 270")
}

// IsAlreadyExists returns true when the API reports the resource already exists (310).
func IsAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "API error 310")
}

