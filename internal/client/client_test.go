package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func apiResponse(code, errmessage string, data interface{}) []byte {
	if data == nil {
		data = map[string]interface{}{}
	}
	raw, _ := json.Marshal([]map[string]interface{}{
		{"code": code, "errmessage": errmessage, "data": data},
	})
	return raw
}

func apiResponseMessage(code, message string, data interface{}) []byte {
	if data == nil {
		data = map[string]interface{}{}
	}
	raw, _ := json.Marshal([]map[string]interface{}{
		{"code": code, "message": message, "data": data},
	})
	return raw
}

func TestClient_Do_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(apiResponse("100", "Action executed successfully", map[string]string{"id": "123"}))
	}))
	defer server.Close()

	c := NewWithEndpoint("test-token", "test", server.URL)
	resp, err := c.Do(context.Background(), Request{"command": "get_hosts"})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp == nil {
		t.Fatal("Do() response is nil")
	}
	if resp.codeString() != "100" {
		t.Errorf("code = %q, want 100", resp.codeString())
	}
	if string(resp.Data) != `{"id":"123"}` {
		t.Errorf("data = %s", resp.Data)
	}
}

func TestClient_Do_ErrorCodes(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		wantNotFound bool
		wantPerm     bool
	}{
		{"210 invalid param", "210", true, false},
		{"220 auth failed", "220", false, true},
		{"270 insufficient rights", "270", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(apiResponse(tt.code, "error message", nil))
			}))
			defer server.Close()

			c := NewWithEndpoint("token", "test", server.URL)
			_, err := c.Do(context.Background(), Request{"command": "test"})
			if err == nil {
				t.Fatal("Do() expected error")
			}
			if got := IsNotFound(err); got != tt.wantNotFound {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.wantNotFound)
			}
			if got := IsPermissionError(err); got != tt.wantPerm {
				t.Errorf("IsPermissionError() = %v, want %v", got, tt.wantPerm)
			}
		})
	}
}

func TestClient_Do_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

	c := NewWithEndpoint("token", "test", server.URL)
	_, err := c.Do(context.Background(), Request{"command": "test"})
	if err == nil {
		t.Fatal("Do() expected error for empty response")
	}
}

func TestClient_Do_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	c := NewWithEndpoint("token", "test", server.URL)
	_, err := c.Do(context.Background(), Request{"command": "test"})
	if err == nil {
		t.Fatal("Do() expected error for invalid JSON")
	}
}

func TestClient_Do_UserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != "terraform-provider-serverguard24/1.2.3" {
			t.Errorf("User-Agent = %q, want %q", ua, "terraform-provider-serverguard24/1.2.3")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(apiResponse("100", "ok", map[string]string{"id": "1"}))
	}))
	defer server.Close()

	c := NewWithEndpoint("token", "1.2.3", server.URL)
	_, _ = c.Do(context.Background(), Request{"command": "test"})
}

func TestClient_Do_TokenInBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Token != "my-secret-token" {
			t.Errorf("token = %q, want %q", body.Token, "my-secret-token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(apiResponse("100", "ok", map[string]string{}))
	}))
	defer server.Close()

	c := NewWithEndpoint("my-secret-token", "test", server.URL)
	_, _ = c.Do(context.Background(), Request{"command": "test"})
}

func TestClient_IsAlreadyExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(apiResponse("310", "Already exists", nil))
	}))
	defer server.Close()

	c := NewWithEndpoint("token", "test", server.URL)
	_, err := c.Do(context.Background(), Request{"command": "test"})
	if err == nil {
		t.Fatal("expected error for 310")
	}
	if !IsAlreadyExists(err) {
		t.Errorf("IsAlreadyExists() = false, want true for: %v", err)
	}
	if IsNotFound(err) {
		t.Error("IsNotFound() should be false for 310")
	}
}

func TestClient_Do_ErrorMessageFields(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{"message", apiResponseMessage("210", "message field", nil), "message field"},
		{"errmessage", apiResponse("210", "errmessage field", nil), "errmessage field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(tt.body)
			}))
			defer server.Close()

			c := NewWithEndpoint("token", "test", server.URL)
			_, err := c.Do(context.Background(), Request{"command": "test"})
			if err == nil {
				t.Fatal("Do() expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.want)
			}
		})
	}
}

func TestClient_Do_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewWithEndpoint("token", "test", server.URL)
	_, err := c.Do(context.Background(), Request{"command": "test"})
	if err == nil {
		t.Fatal("Do() expected error for non-200 status")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status 500, got: %v", err)
	}
}
