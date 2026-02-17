package bencode

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
)

// Given a value of type int, string, []interface{}, or map[string]interface{},
// encode it in bencode format. Otherwise, return an error.
func Encode(v interface{}) ([]byte, error) {
	switch t := v.(type) {
	case int:
		return encodeInteger(v.(int))
	case string:
		return encodeString(v.(string))
	case []interface{}:
		return encodeList(v.([]interface{}))
	case map[string]interface{}:
		return encodeDictionary(v.(map[string]interface{}))
	default:
		return nil, fmt.Errorf("Invalid type: %v", t)
	}

}

// Given a string, encode it in bencode format.
// E.g. "spam" -> "4:spam"
func encodeString(s string) ([]byte, error) {
	return []byte(strconv.Itoa(len(s)) + ":" + s), nil
}

// Given an integer, encode it in bencode format.
// E.g. 42 -> "i42e"
func encodeInteger(i int) ([]byte, error) {
	return []byte("i" + strconv.Itoa(i) + "e"), nil
}

// Given a list, encode it in bencode format.
// E.g. []interface{}{"spam", 42} -> "l4:spami42ee"
func encodeList(l []interface{}) ([]byte, error) {
	parts := [][]byte{[]byte("l")}

	for _, v := range l {
		encoded, _ := Encode(v)
		parts = append(parts, encoded)
	}

	parts = append(parts, []byte("e"))
	return bytes.Join(parts, []byte{}), nil
}

// Given a dictionary, encode it in bencode format.
// E.g. map[string]interface{}{"foo": "bar", "baz": 42} -> "d3:bar3:bazi3:foo3e"
func encodeDictionary(d map[string]interface{}) ([]byte, error) {
	parts := [][]byte{[]byte("d")}

	keys := []string{}
	for k := range d {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		encodedKey, _ := encodeString(k)
		encodedValue, _ := Encode(d[k])
		parts = append(parts, encodedKey, encodedValue)
	}

	parts = append(parts, []byte("e"))
	return bytes.Join(parts, []byte{}), nil
}
