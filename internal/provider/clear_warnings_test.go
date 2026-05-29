package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWarnIfClearingString(t *testing.T) {
	tests := []struct {
		name  string
		state types.String
		plan  types.String
		want  int
	}{
		{
			name:  "value to null warns",
			state: types.StringValue("old"),
			plan:  types.StringNull(),
			want:  1,
		},
		{
			name:  "value to empty warns",
			state: types.StringValue("old"),
			plan:  types.StringValue(""),
			want:  1,
		},
		{
			name:  "value to value does not warn",
			state: types.StringValue("old"),
			plan:  types.StringValue("new"),
		},
		{
			name:  "empty state does not warn",
			state: types.StringValue(""),
			plan:  types.StringNull(),
		},
		{
			name:  "null state does not warn",
			state: types.StringNull(),
			plan:  types.StringNull(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diags diag.Diagnostics
			warnIfClearingString(&diags, path.Root("field"), tt.state, tt.plan)
			if got := diags.WarningsCount(); got != tt.want {
				t.Fatalf("WarningsCount() = %d, want %d", got, tt.want)
			}
			assertNoErrors(t, diags)
		})
	}
}

func TestWarnIfClearingBool(t *testing.T) {
	tests := []struct {
		name  string
		state types.Bool
		plan  types.Bool
		want  int
	}{
		{
			name:  "true to false warns",
			state: types.BoolValue(true),
			plan:  types.BoolValue(false),
			want:  1,
		},
		{
			name:  "true to null warns",
			state: types.BoolValue(true),
			plan:  types.BoolNull(),
			want:  1,
		},
		{
			name:  "true to true does not warn",
			state: types.BoolValue(true),
			plan:  types.BoolValue(true),
		},
		{
			name:  "false to false does not warn",
			state: types.BoolValue(false),
			plan:  types.BoolValue(false),
		},
		{
			name:  "null state does not warn",
			state: types.BoolNull(),
			plan:  types.BoolValue(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diags diag.Diagnostics
			warnIfClearingBool(&diags, path.Root("field"), tt.state, tt.plan)
			if got := diags.WarningsCount(); got != tt.want {
				t.Fatalf("WarningsCount() = %d, want %d", got, tt.want)
			}
			assertNoErrors(t, diags)
		})
	}
}

func TestWarnIfClearingParamsWarnsForAllStringLikeParams(t *testing.T) {
	state := &CheckParamsModel{
		Path:       types.StringValue("/health"),
		Login:      types.StringValue("user"),
		SslVersion: types.StringValue("tls12"),
		Ssl:        types.BoolValue(true),
		Timeout:    types.Int64Value(10),
	}
	plan := &CheckParamsModel{
		Path:       types.StringNull(),
		Login:      types.StringValue("new-user"),
		SslVersion: types.StringNull(),
		Ssl:        types.BoolValue(false),
		Timeout:    types.Int64Null(),
	}

	var diags diag.Diagnostics
	warnIfClearingParams(&diags, state, plan)
	assertNoErrors(t, diags)

	wantPaths := map[string]bool{
		"params.path":        true,
		"params.ssl_version": true,
		"params.ssl":         true,
	}
	assertWarningPaths(t, diags, wantPaths)
}

func TestWarnIfClearingParamsHandlesRemovedParamsBlock(t *testing.T) {
	state := &CheckParamsModel{
		Path: types.StringValue("/health"),
		Ssl:  types.BoolValue(true),
	}

	var diags diag.Diagnostics
	warnIfClearingParams(&diags, state, nil)
	assertNoErrors(t, diags)

	wantPaths := map[string]bool{
		"params.path": true,
		"params.ssl":  true,
	}
	assertWarningPaths(t, diags, wantPaths)
}

func assertNoErrors(t *testing.T, diags diag.Diagnostics) {
	t.Helper()
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
}

func assertWarningPaths(t *testing.T, diags diag.Diagnostics, want map[string]bool) {
	t.Helper()
	if got := diags.WarningsCount(); got != len(want) {
		t.Fatalf("WarningsCount() = %d, want %d; warnings: %v", got, len(want), diags.Warnings())
	}
	for _, d := range diags.Warnings() {
		withPath, ok := d.(diag.DiagnosticWithPath)
		if !ok {
			t.Fatalf("warning %q has no path", d.Summary())
		}
		gotPath := withPath.Path().String()
		if !want[gotPath] {
			t.Fatalf("unexpected warning path %q", gotPath)
		}
		delete(want, gotPath)
	}
	if len(want) > 0 {
		t.Fatalf("missing warning paths: %v", want)
	}
}
