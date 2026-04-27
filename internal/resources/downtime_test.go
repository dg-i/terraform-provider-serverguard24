package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStringListToInt64s(t *testing.T) {
	list := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("123"),
		types.StringValue("abc"),
		types.StringValue("0"),
		types.StringValue("456"),
	})

	got, invalid := stringListToInt64s(context.Background(), list)
	if len(got) != 2 || got[0] != 123 || got[1] != 456 {
		t.Fatalf("stringListToInt64s() valid IDs = %v, want [123 456]", got)
	}
	if len(invalid) != 2 || invalid[0] != "abc" || invalid[1] != "0" {
		t.Fatalf("stringListToInt64s() invalid IDs = %v, want [abc 0]", invalid)
	}
}
