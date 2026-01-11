package bencode

import (
	"reflect"
	"testing"
)

func TestEncodeString(t *testing.T) {
	strings := map[string][]byte{
		"spam":        []byte("4:spam"),         // Short string
		"":            []byte("0:"),             // Empty string
		"hello world": []byte("11:hello world"), // String with space
	}

	for input, expected := range strings {
		encoded, err := encodeString(input)

		if err != nil {
			t.Errorf("Unexpected error encoding string %q: %v", input, err)
		}

		if !reflect.DeepEqual(expected, encoded) {
			t.Errorf("Encoding string %q: expected %q, got %q", input, expected, encoded)
		}
	}
}

func TestEncodeInteger(t *testing.T) {
	integers := map[int][]byte{
		42:    []byte("i42e"),    // Positive integer
		0:     []byte("i0e"),     // Zero
		-7:    []byte("i-7e"),    // Negative integer
		12345: []byte("i12345e"), // Larger integer
	}

	for input, expected := range integers {
		encoded, err := encodeInteger(input)

		if err != nil {
			t.Errorf("Unexpected error encoding integer %d: %v", input, err)
		}

		if !reflect.DeepEqual(expected, encoded) {
			t.Errorf("Encoding integer %d: expected %q, got %q", input, expected, encoded)
		}
	}
}

func TestEncodeList(t *testing.T) {
	type ListTest struct {
		input    []interface{}
		expected []byte
	}

	lists := []ListTest{
		{[]interface{}{"spam", 42}, []byte("l4:spami42ee")},                                      // Mixed types
		{[]interface{}{}, []byte("le")},                                                          // Empty list
		{[]interface{}{"hello", "world"}, []byte("l5:hello5:worlde")},                            // List of strings
		{[]interface{}{1, 2, 3, 4, 5}, []byte("li1ei2ei3ei4ei5ee")},                              // List of integers
		{[]interface{}{"nested", []interface{}{1, "two", 3}}, []byte("l6:nestedli1e3:twoi3eee")}, // Nested list
		{[]interface{}{map[string]interface{}{"key": "val"}}, []byte("ld3:key3:valee")},          // List with dictionary
	}

	for _, list := range lists {
		encoded, err := encodeList(list.input)

		if err != nil {
			t.Errorf("Unexpected error encoding list %v: %v", list.input, err)
		}

		if !reflect.DeepEqual(list.expected, encoded) {
			t.Errorf("Encoding list %v: expected %q, got %q", list.input, list.expected, encoded)
		}
	}
}

func TestEncodeDictionary(t *testing.T) {
	type DictTest struct {
		input    map[string]interface{}
		expected []byte
	}

	dictionaries := []DictTest{
		{map[string]interface{}{"foo": "bar", "baz": 42}, []byte("d3:bazi42e3:foo3:bare")},                     // Mixed types
		{map[string]interface{}{}, []byte("de")},                                                               // Empty dictionary
		{map[string]interface{}{"key1": "val1", "key2": "val2"}, []byte("d4:key14:val14:key24:val2e")},         // String values
		{map[string]interface{}{"num1": 1, "num2": 2, "num3": 3}, []byte("d4:num1i1e4:num2i2e4:num3i3ee")},     // Integer values
		{map[string]interface{}{"list": []interface{}{"a", "b", "c"}}, []byte("d4:listl1:a1:b1:cee")},          // Nested list
		{map[string]interface{}{"dict": map[string]interface{}{"key": "val"}}, []byte("d4:dictd3:key3:valee")}, // Nested dictionary
	}

	for _, dict := range dictionaries {
		encoded, err := encodeDictionary(dict.input)

		if err != nil {
			t.Errorf("Unexpected error encoding dictionary %v: %v", dict.input, err)
		}

		if !reflect.DeepEqual(dict.expected, encoded) {
			t.Errorf("Encoding dictionary %v: expected %q, got %q", dict.input, dict.expected, encoded)
		}
	}
}
