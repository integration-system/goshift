package goshift

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestManualShifter_ApplyNestedReporter(t *testing.T) {
	mapping := map[string][]string{
		"root.lvl1.lvl2.lvl3": {"replaced.lvl3_mapped"},
		"root.lvl1":           {"replaced.lvl1_mapped"},
	}
	shifter, err := NewShifterV2(mapping)
	if err != nil {
		t.Fatal(err)
	}

	sourceStr := `{ "root": {"lvl1": {}} }`
	sourceMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(sourceStr), &sourceMap)
	if err != nil {
		t.Fatal(err)
	}

	reporter := func(src string, dst string, value interface{}) interface{} {
		if src == "root.lvl1.lvl2.lvl3" {
			return "lvl3_mapped_replaced"
		}
		return value
	}

	result, err := shifter.Apply(sourceMap, WithReporter(reporter))
	if err != nil {
		t.Fatal(err)
	}

	expectedResult := map[string]interface{}{
		"replaced": map[string]interface{}{
			"lvl3_mapped": "lvl3_mapped_replaced",
			"lvl1_mapped": map[string]interface{}{},
		},
	}

	if !reflect.DeepEqual(result, expectedResult) {
		t.Fail()
	}
}
