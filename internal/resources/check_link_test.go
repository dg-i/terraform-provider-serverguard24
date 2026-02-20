package resources

import (
	"testing"
)

func TestParseLinkID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		wantA     string
		wantB     string
		wantDiag  bool
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
