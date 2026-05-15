package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The Serverguard24 API is asymmetric on edit:
//
//   - Bool/int flags: sending the field as 0 returns HTTP 500, so the provider
//     omits the field instead. The API then keeps the previous value, so
//     toggling a flag from true → false in config has no effect at apply time.
//   - String fields: passing an empty string on edit is treated by the API as
//     "no change" — clearing a previously-set value silently fails.
//
// The helpers below emit a plan-time warning when a user attempts either
// operation, so the user sees the issue at plan time instead of staring at a
// perpetual diff after apply.

const apiClearLimitationDetail = "The Serverguard24 API does not accept a clear/false value on edit — the previous value will remain. To force the change, either taint the resource (`terraform taint` / `tofu taint`) to recreate it, or update the field via the Serverguard24 dashboard."

func warnIfClearingString(diags *diag.Diagnostics, p path.Path, state, plan types.String) {
	if state.IsNull() || state.IsUnknown() || state.ValueString() == "" {
		return
	}
	if !plan.IsNull() && !plan.IsUnknown() && plan.ValueString() != "" {
		return
	}
	diags.AddAttributeWarning(p,
		"Clearing this field has no effect",
		apiClearLimitationDetail)
}

func warnIfClearingBool(diags *diag.Diagnostics, p path.Path, state, plan types.Bool) {
	if state.IsNull() || state.IsUnknown() || !state.ValueBool() {
		return
	}
	if !plan.IsNull() && !plan.IsUnknown() && plan.ValueBool() {
		return
	}
	diags.AddAttributeWarning(p,
		"Disabling this flag has no effect",
		apiClearLimitationDetail)
}
