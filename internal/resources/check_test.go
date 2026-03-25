package resources

import (
	"testing"
)

func TestCheckTypeByName_AllMapped(t *testing.T) {
	if len(checkTypeByName) == 0 {
		t.Fatal("checkTypeByName is empty")
	}
	for name, id := range checkTypeByName {
		if id <= 0 {
			t.Errorf("checkTypeByName[%q] = %d, want positive integer", name, id)
		}
	}
}

func TestCheckTypeByID_Roundtrip(t *testing.T) {
	for name, id := range checkTypeByName {
		got, ok := checkTypeByID[id]
		if !ok {
			t.Errorf("checkTypeByID[%d] not found (expected %q)", id, name)
			continue
		}
		if got != name {
			t.Errorf("checkTypeByID[%d] = %q, want %q", id, got, name)
		}
	}
}

func TestCheckTypeByID_SameSize(t *testing.T) {
	if len(checkTypeByName) != len(checkTypeByID) {
		t.Errorf("len(checkTypeByName) = %d, len(checkTypeByID) = %d, want equal (duplicate IDs?)",
			len(checkTypeByName), len(checkTypeByID))
	}
}

func TestValidCheckTypes_Sorted(t *testing.T) {
	names := validCheckTypes()
	if len(names) == 0 {
		t.Fatal("validCheckTypes() returned empty slice")
	}
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("validCheckTypes() not sorted: %q comes after %q", names[i], names[i-1])
		}
	}
}

func TestParseRFC3339Unix(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"UTC midnight", "2026-04-01T00:00:00Z", 1775001600, false},
		{"UTC with hours", "2026-04-01T02:00:00Z", 1775008800, false},
		{"with timezone offset", "2026-04-01T02:00:00+02:00", 1775001600, false},
		{"invalid format", "2026-04-01 00:00:00", 0, true},
		{"empty", "", 0, true},
		{"date only", "2026-04-01", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRFC3339Unix(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRFC3339Unix(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseRFC3339Unix(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
