package bencode

import (
	"reflect"
	"testing"
)

func TestValidIntDecode(t *testing.T) {
	var validInts = map[string][2]int{
		"i128e": {5, 128}, // normal positive integer
		"i0e":   {3, 0},   // zero
		"i-3e":  {4, -3},  // negative integer
	}

	for i, expected := range validInts {
		expectedIndex := expected[0]
		expectedInt := expected[1]
		encodedInt := []byte(i)

		val, index, err := Decode(encodedInt)

		if err != nil {
			t.Errorf("Error decoding %s.\n%v", encodedInt, err)
		}

		if index != expectedIndex {
			t.Errorf("The given index is %v, the expected index is %v", index, expectedIndex)
		}

		if val != expectedInt {
			t.Errorf("The given int is %v, the expected int is %v", val, expectedInt)
		}
	}
}

func TestInvalidIntDecode(t *testing.T) {
	var invalidInts = [][]byte{
		[]byte("i12"),  // missing terminator
		[]byte("i-0e"), // negative zero
		[]byte("i03e"), // leading zero
		[]byte("ie"),   // empty integer
	}

	for _, encodedInt := range invalidInts {
		_, _, err := Decode(encodedInt)

		if err == nil {
			t.Errorf("Expected error decoding invalid int %s, but got none", encodedInt)
		}
	}
}

func TestValidStringDecode(t *testing.T) {
	var validStrings = map[string][2]interface{}{
		"4:spam":        {6, "spam"},        // normal string
		"0:":            {2, ""},            // empty string
		"10:hello worl": {13, "hello worl"}, // longer string
	}

	for i, expected := range validStrings {
		expectedIndex := expected[0].(int)
		expectedString := expected[1].(string)
		encodedString := []byte(i)

		val, index, err := Decode(encodedString)

		if err != nil {
			t.Errorf("Error decoding %s.\n%v", encodedString, err)
		}

		if index != expectedIndex {
			t.Errorf("The given index is %v, the expected index is %v", index, expectedIndex)
		}

		if val != expectedString {
			t.Errorf("The given string is %v, the expected string is %v", val, expectedString)

		}
	}
}

func TestInvalidStringDecode(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected code to panic, got none")
		}
	}()

	var invalidStrings = [][]byte{
		[]byte("4spam"), // missing colon
		[]byte("3:ab"),  // length shorter than actual string
		[]byte("2:a"),   // length longer than actual string
		[]byte("-1:a"),  // negative length
		[]byte("a:a"),   // non-integer length
	}

	for _, encodedString := range invalidStrings {
		_, _, err := Decode(encodedString)
		if err == nil {
			t.Errorf("Expected error decoding invalid string %s, but got none", encodedString)
		}
	}
}

func TestValidListDecode(t *testing.T) {
	var validLists = map[string][2]interface{}{
		"l4:spame":                 {8, []interface{}{"spam"}},                                          // Single string element
		"l4:spam3:bage":            {13, []interface{}{"spam", "bag"}},                                  // Multiple string elements
		"li1ee":                    {5, []interface{}{1}},                                               // Single integer element
		"li1ei2ei3ee":              {11, []interface{}{1, 2, 3}},                                        // Multiple integer elements
		"le":                       {2, []interface{}{}},                                                // Empty list
		"lli1ei2ei3eeli1ei2ei3eee": {24, []interface{}{[]interface{}{1, 2, 3}, []interface{}{1, 2, 3}}}, // Nested Lists
		"l4:spamli1ei2ee3:bage":    {21, []interface{}{"spam", []interface{}{1, 2}, "bag"}},             // Mixed element types
		"ld3:key3:valee":           {14, []interface{}{map[string]interface{}{"key": "val"}}},           // List with dictionary
	}

	for i, expected := range validLists {
		expectedIndex := expected[0].(int)
		expectedList := expected[1]
		encodedList := []byte(i)

		val, index, err := Decode(encodedList)

		if err != nil {
			t.Errorf("Error decoding %s.\n%v", encodedList, err)
		}

		if index != expectedIndex {
			t.Errorf("The given index is %v, the expected index is %v", index, expectedIndex)
		}

		if !reflect.DeepEqual(expectedList, val) {
			t.Errorf("The given list is %v, the expected list is %v", val, expectedList)
		}
	}
}

func TestInvalidListDecode(t *testing.T) {
	var invalidLists = [][]byte{
		[]byte("l4:spami2e"),   // missing terminator
		[]byte("li1ei2e"),      // missing terminator
		[]byte("l4:spam3:bag"), // missing terminator
	}

	for _, encodedList := range invalidLists {
		_, _, err := Decode(encodedList)
		if err == nil {
			t.Errorf("Expected error decoding invalid list %s, but got none", encodedList)
		}
	}
}

func TestValidDictionaryDecode(t *testing.T) {
	var validDicts = map[string][2]interface{}{
		"d4:spaml1:a1:bee":        {16, map[string]interface{}{"spam": []interface{}{"a", "b"}}},              // Single string key-value pair
		"d3:bar3:baz3:foo3:quee":  {22, map[string]interface{}{"bar": "baz", "foo": "que"}},                   // Multiple string key-value pairs
		"d3:fooi42ee":             {11, map[string]interface{}{"foo": 42}},                                    // Single integer value
		"d3:barli1ei2ee3:fooi3ee": {23, map[string]interface{}{"bar": []interface{}{1, 2}, "foo": 3}},         // Mixed value types
		"de":                      {2, map[string]interface{}{}},                                              // Empty dictionary
		"d4:dictd3:key3:valee":    {20, map[string]interface{}{"dict": map[string]interface{}{"key": "val"}}}, // Nested dictionary
	}

	for i, expected := range validDicts {
		expectedIndex := expected[0].(int)
		expectedDict := expected[1]
		encodedDict := []byte(i)

		val, index, err := Decode(encodedDict)

		if err != nil {
			t.Errorf("Error decoding %s.\n%v", encodedDict, err)
		}

		if index != expectedIndex {
			t.Errorf("The given index is %v, the expected index is %v", index, expectedIndex)
		}

		if !reflect.DeepEqual(expectedDict, val) {
			t.Errorf("The given dict is %v, the expected dict is %v", val, expectedDict)
		}
	}
}

func TestInvalidDictionaryDecode(t *testing.T) {
	var invalidDicts = [][]byte{
		[]byte("d3:bar3:baz"),            // missing terminator
		[]byte("di1ei2ee"),               // non-string key
		[]byte("d3:foo3:bar3:foo3:baze"), // duplicate key
		[]byte("d3:foo3:bar3:abc1:ae"),   // keys not in sorted order
	}

	for _, encodedDict := range invalidDicts {
		_, _, err := Decode(encodedDict)
		if err == nil {
			t.Errorf("Expected error decoding invalid dict %s, but got none", encodedDict)
		}
	}
}

func TestInvalidDecodeFormat(t *testing.T) {
	_, _, err := Decode([]byte("x4:spam"))
	if err == nil {
		t.Errorf("Expected error encoding invalid value x4:spam, but got none")
	}
}
