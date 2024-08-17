package utils

import "encoding/json"

func DeepCopy[T any](input T) (T, error) {
	// Marshal the input object to JSON
	data, err := json.Marshal(input)
	if err != nil {
		return *new(T), err
	}

	// Unmarshal the JSON back into a new object of type T
	var output T
	err = json.Unmarshal(data, &output)
	if err != nil {
		return *new(T), err
	}

	return output, nil
}
