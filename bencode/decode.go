package bencode

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/mitchellh/mapstructure"
)

// Decode takes in a byte array of bencoded data and decodes it into the provided info struct.
func Decode(data []byte, info interface{}) error {
	extractedData, _, err := doDecode(data)
	if err != nil {
		return err
	}

	err = mapstructure.Decode(extractedData, info)
	if err != nil {
		return err
	}

	return nil

}

///////////////////////////// Helper functions /////////////////////////////

// Takes in a byte array of bencoded data and decodes it into the appropriate Go data type
// (int, string, list, or dictionary) and returns the decoded value, the number of bytes read
// and any error encountered.
func doDecode(data []byte) (interface{}, int, error) {
	switch firstChar := data[0]; {
	case firstChar == 'i':
		return decodeInteger(data)
	case firstChar == 'l':
		return decodeList(data)
	case firstChar == 'd':
		return decodeDictionary(data)
	case firstChar >= '0' && firstChar <= '9':
		return decodeString(data)
	}

	return nil, 0, fmt.Errorf("Error: Data has invalid format, it must start with 'i', 'l', 'd', or a number between 0-9, %v", data)
}

// Given a bencoded integer at the start of the data, decode it
// and return the decoded integer, the number of bytes read, and any error encountered.
// E.g. "i42e" -> 42
func decodeInteger(data []byte) (int, int, error) {
	i := 0
	if data[1] == '-' {
		i = 1
	}

	start := i
	end := 0
	for ; i < len(data); i++ {
		if data[i] == 'e' {
			end = i
			break
		}
	}

	if end == 0 {
		return 0, 0, missingTerminatorError("integer")
	}

	decodedInt, err := strconv.Atoi(string(data[1:end]))
	if err != nil {
		return 0, 0, err
	}

	if data[start+1] == '0' && start+1 != end-1 {
		return 0, 0, fmt.Errorf("Error: Leading zeros are not allowed")
	}

	if decodedInt == 0 && start == 1 {
		return 0, 0, fmt.Errorf("Error: -0 is invalid")
	}

	return decodedInt, end + 1, nil
}

// Decodes a string from the bencoded data and returns
// the decoded string, the number of bytes read, and any error encountered.
// E.g. "4:spam" -> "spam"
func decodeString(data []byte) (string, int, error) {
	end := 0
	for i := 0; i < len(data); i++ {
		if data[i] == ':' {
			end = i
			break
		}
	}

	if end == 0 {
		return "", 0, fmt.Errorf("Error: No ':' character found in %v", data)
	}

	decodedStrLength, err := checkVariableLength(data[0:end], "string")

	if err != nil {
		return "", 0, err
	}

	decodedStr := string(data[end+1 : end+1+decodedStrLength])

	return decodedStr, end + decodedStrLength + 1, nil
}

// Decodes a list from the bencoded data and returns
// the decoded list, the number of bytes read, and any error encountered.
// E.g. "l4:spami42ee" -> []interface{}{"spam", 42}
func decodeList(data []byte) ([]interface{}, int, error) {
	decodedList := []interface{}{}

	next := 1
	end := 0
	for next < len(data) {
		// Check for end of list
		if data[next] == 'e' {
			end = next
			break
		}

		decodedVal, index, err := doDecode(data[next:])

		if err != nil {
			return decodedList, 0, err
		}

		decodedList = append(decodedList, decodedVal)
		next += index
	}

	if end == 0 {
		return decodedList, 0, missingTerminatorError("list")
	}

	return decodedList, next + 1, nil
}

// Decodes a dictionary from the bencoded data and returns
// the decoded dictionary, the number of bytes read, and any error encountered.
// E.g. "d3:bar3:bazi3:foo3e" -> map[string]interface{}{"foo": "bar", "baz": 42}
func decodeDictionary(data []byte) (map[string]interface{}, int, error) {
	decodedMap := map[string]interface{}{}
	keys := []string{}

	next := 1
	end := 0

	for next < len(data) {
		if data[next] == 'e' {
			end = next
			break
		}

		extractedKey, index, err := doDecode(data[next:])
		if err != nil {
			return decodedMap, 0, err
		}

		key, ok := extractedKey.(string)
		if !ok {
			return decodedMap, 0, fmt.Errorf("Error: Key is not a string, %v", extractedKey)
		}

		next += index

		_, ok = decodedMap[key]
		if ok {
			return decodedMap, 0, fmt.Errorf("Error: Duplicate key in map")
		}

		val, index, err := doDecode(data[next:])

		if err != nil {
			return decodedMap, 0, err
		}

		next += index

		decodedMap[key] = val
		keys = append(keys, key)
	}

	if end == 0 {
		return decodedMap, 0, missingTerminatorError("dictionary")
	}

	// Checking if keys not sorted
	if !slices.IsSorted(keys) {
		return decodedMap, 0, fmt.Errorf("Error: Keys are not sorted in ascending order")
	}

	return decodedMap, next + 1, nil

}

// Checks if the variable length is valid checking that it is an integer and that it is not negative.
// Returns the variable length as an integer and any error encountered.
func checkVariableLength(variableLength []byte, variableType string) (int, error) {
	length, err := strconv.Atoi(string(variableLength))
	if err != nil {
		return 0, err
	}

	if length < 0 {
		return 0, fmt.Errorf("Error: Encoded %s has negative length, %v", variableType, length)
	}

	return length, nil
}

// Wrapper function for missing terminator error to specify the variable type in the error message.
func missingTerminatorError(variableType string) error {
	return fmt.Errorf("Error: Missing terminator for %s", variableType)
}
