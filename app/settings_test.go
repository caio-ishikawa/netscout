package app

import (
	"reflect"
	"testing"
)

func TestParseKeyValStr(t *testing.T) {
	cases := map[string]struct {
		input  string
		result map[string]string
		err    bool
	}{
		"empty": {
			input:  "",
			result: map[string]string{},
			err:    true,
		},
		"singleKeyValue": {
			input:  "test=value",
			result: map[string]string{"test": "value"},
			err:    false,
		},
		"multipleKeyValues": {
			input:  "test=value,another=test",
			result: map[string]string{"test": "value", "another": "test"},
			err:    false,
		},
		"invalidStr": {
			input:  "invalid",
			result: map[string]string{},
			err:    true,
		},
	}

	for name, tc := range cases {
		res, err := parseKeyValueStr(tc.input)
		if tc.err {
			if err == nil {
				t.Errorf("%s expected error but got none", name)
			}
		}

		if !reflect.DeepEqual(tc.result, res) {
			t.Errorf("%s expected %s but got %s", name, tc.result, res)
		}
	}
}
