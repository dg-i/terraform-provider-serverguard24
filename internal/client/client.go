package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultAPIEndpoint = "https://www.serverguard24.de/de/api/json.html"

// Client is the Serverguard24 API client.
type Client struct {
	token      string
	endpoint   string
	httpClient *http.Client
}

// New creates a new API client using the default Serverguard24 API endpoint.
func New(token string) *Client {
	return NewWithEndpoint(token, defaultAPIEndpoint)
}

// NewWithEndpoint creates a new API client with a custom endpoint (e.g. for testing).
func NewWithEndpoint(token, endpoint string) *Client {
	return &Client{
		token:      token,
		endpoint:   endpoint,
		httpClient: &http.Client{},
	}
}

// Request is a single command in the batch request body.
type Request map[string]interface{}

// Response is a single response element returned by the API.
// The API may return "code" as either a JSON string ("100") or a number (100).
type Response struct {
	Code       json.RawMessage `json:"code"`
	ErrMessage string          `json:"errmessage"`
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

	body := map[string]interface{}{
		"token":    c.token,
		"requests": []Request{req},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	endpoint := defaultAPIEndpoint
	if c.endpoint != "" {
		endpoint = c.endpoint
	}
	resp, err := c.httpClient.Post(endpoint, "application/json", bytes.NewReader(payload))
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

// IsNotFound returns true when the error represents a missing resource.
// The API returns code 210 (invalid parameter) when an ID doesn't exist.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	// API returns 210 for invalid/unknown IDs
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

