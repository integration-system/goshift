package goshift

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func TestManualShifter_ApplyNestedReporter(t *testing.T) {
	mapping := map[string][]string{
		"root.lvl1.lvl2.lvl3": {"replaced.lvl3_mapped"},
		"root.lvl1":           {"replaced.lvl1_mapped"},
		"root.arr1[].key":     {"replaced.arr1[].key"},
		"root.arr1[].key2":    {"replaced.arr1[].key2"},
		"root.lvl1.imaginaryArr2[].struct.subval1": {"replaced.arr2[].val1"},
		"root.lvl1.imaginaryArr2[].struct.subval2": {"replaced.arr2[].val2"},
	}
	shifter, err := NewShifterV2(mapping)
	if err != nil {
		t.Fatal(err)
	}

	sourceStr := `{ "root": {"lvl1": {}, "arr1": [{"key":"1"}, {"key":"2"}, {"key":"3"}] }}`
	sourceMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(sourceStr), &sourceMap)
	if err != nil {
		t.Fatal(err)
	}

	reportedSrc := make([]string, 0)
	reporter := func(src string, dst string, value interface{}) interface{} {
		reportedSrc = append(reportedSrc, src)
		if src == "root.lvl1.lvl2.lvl3" && value == nil {
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
			"arr1": []interface{}{
				map[string]interface{}{"key": "1"},
				map[string]interface{}{"key": "2"},
				map[string]interface{}{"key": "3"},
			},
		},
	}
	expectedReportedSrc := []string{
		"root.lvl1",
		"root.arr1[].key",
		"root.arr1[].key",
		"root.arr1[].key",
		"root.arr1[].key2",
		"root.arr1[].key2",
		"root.arr1[].key2",
		"root.lvl1.lvl2.lvl3",
	}
	sort.Strings(reportedSrc)
	sort.Strings(expectedReportedSrc)

	if !reflect.DeepEqual(result, expectedResult) {
		resJson, _ := json.Marshal(result)
		expResJson, _ := json.Marshal(expectedResult)
		fmt.Printf("result:\n%s\n\nexpected:\n%s", resJson, expResJson)
		t.Fail()
	}
	if !reflect.DeepEqual(reportedSrc, expectedReportedSrc) {
		t.Fail()
	}
}

func Test_compilePair(t *testing.T) {
	type args struct {
		srcPath string
		dstPath string
	}
	tests := []struct {
		name    string
		args    args
		want    []term
		wantErr bool
	}{
		{
			name: "srcParts == dstParts; with array", args: args{
				srcPath: "lvl1.lvl2[].name",
				dstPath: "new_lvl1.new_lvl2[].new_name",
			}, want: []term{
				newTerm("lvl1", false, []string{"new_lvl1"}),
				newTerm("lvl2", true, []string{"new_lvl2"}),
				newTerm("name", false, []string{"new_name"}),
			}, wantErr: false,
		},
		{
			name: "srcParts == dstParts; without array", args: args{
				srcPath: "lvl1.lvl2",
				dstPath: "new_lvl1.new_lvl2",
			}, want: []term{
				newTerm("lvl1", false, []string{"new_lvl1"}),
				newTerm("lvl2", false, []string{"new_lvl2"}),
			}, wantErr: false,
		},
		{
			name: "srcParts >= dstParts; with array", args: args{
				srcPath: "lvl1.lvl2.lvl3arr[].name",
				dstPath: "new_lvl1[].new_name",
			}, want: []term{
				newTerm("lvl1", false, []string{}),
				newTerm("lvl2", false, []string{}),
				newTerm("lvl3arr", true, []string{"new_lvl1"}),
				newTerm("name", false, []string{"new_name"}),
			}, wantErr: false,
		},
		{
			name: "srcParts >= dstParts; without array", args: args{
				srcPath: "lvl1.lvl2.lvl3",
				dstPath: "new_lvl1.new_lvl2",
			}, want: []term{
				newTerm("lvl1", false, []string{"new_lvl1"}),
				newTerm("lvl2", false, []string{}),
				newTerm("lvl3", false, []string{"new_lvl2"}),
			}, wantErr: false,
		},
		{
			name: "srcParts <= dstParts; with array", args: args{
				srcPath: "lvl1.lvl2arr[].name",
				dstPath: "new_lvl1.new_lvl2.new_lvl3[].new_name",
			}, want: []term{
				newTerm("lvl1", false, []string{"new_lvl1", "new_lvl2"}),
				newTerm("lvl2arr", true, []string{"new_lvl3"}),
				newTerm("name", false, []string{"new_name"}),
			}, wantErr: false,
		},
		{
			name: "srcParts <= dstParts; without array", args: args{
				srcPath: "lvl1.lvl2",
				dstPath: "new_lvl1.new_lvl2.new_lvl3.new_lvl4",
			}, want: []term{
				newTerm("lvl1", false, []string{"new_lvl1"}),
				newTerm("lvl2", false, []string{"new_lvl2", "new_lvl3", "new_lvl4"}),
			}, wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compilePair(tt.args.srcPath, tt.args.dstPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("compilePair() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("compilePair() got = %v, want %v", got, tt.want)
			}
		})
	}
}
