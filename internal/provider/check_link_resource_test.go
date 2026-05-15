package provider

import (
	"testing"
)

func TestParseLinkID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		wantA    string
		wantB    string
		wantDiag bool
	}{
		{"valid simple", "123/456", "123", "456", false},
		{"valid with spaces", " 123 / 456 ", "123", "456", false},
		{"empty", "", "", "", true},
		{"no slash", "123", "", "", true},
		{"empty first segment", "/456", "", "", true},
		{"empty second segment", "123/", "", "", true},
		{"three segments", "a/b/c", "", "", true},
		{"multiple slashes", "1/2/3/4", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotA, gotB, diags := parseLinkID(tt.id, "test_resource")
			hasDiag := diags.HasError()
			if hasDiag != tt.wantDiag {
				t.Errorf("parseLinkID() hasDiag = %v, wantDiag %v; diags: %s", hasDiag, tt.wantDiag, diags)
				return
			}
			if !tt.wantDiag {
				if gotA != tt.wantA || gotB != tt.wantB {
					t.Errorf("parseLinkID() = %q, %q, want %q, %q", gotA, gotB, tt.wantA, tt.wantB)
				}
			}
		})
	}
}

func TestParsePositiveID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		want    int64
		wantErr bool
	}{
		{"valid", "123", 123, false},
		{"zero", "0", 0, true},
		{"negative", "-1", 0, true},
		{"empty", "", 0, true},
		{"non numeric", "abc", 0, true},
		{"with spaces", " 123 ", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePositiveID(tt.id, "id")
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePositiveID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parsePositiveID() = %d, want %d", got, tt.want)
			}
		})
	}
}
