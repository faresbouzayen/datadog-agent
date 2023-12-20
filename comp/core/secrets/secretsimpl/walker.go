// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package secretsimpl

import (
	"fmt"
	"strconv"
)

type resolveFunc func([]string, string) (string, error)

type walker struct {
	// resolver is called to fetch the value of a handle
	resolver resolveFunc
}

// string handles string types, calling the resolver and returning the value to replace the original string with.
func (w *walker) string(text string, yamlPath []string) (string, error) {
	newValue, err := w.resolver(yamlPath, text)
	if err != nil {
		return text, err
	}
	return newValue, err
}

// slice handles slice types, the walker will recursively explore each element of the slice continuing its search for
// strings to replace.
func (w *walker) slice(currentSlice []interface{}, yamlPath []string) error {
	for idx, k := range currentSlice {

		path := make([]string, len(yamlPath), len(yamlPath)+1)
		copy(path, yamlPath)
		path = append(path, strconv.Itoa(idx))

		switch v := k.(type) {
		case string:
			newValue, err := w.string(v, path)
			if err != nil {
				return err
			}
			currentSlice[idx] = newValue
		case map[interface{}]interface{}:
			if err := w.hash(v, path); err != nil {
				return err
			}
		case []interface{}:
			if err := w.slice(v, path); err != nil {
				return err
			}
		}
	}
	return nil
}

// hash handles map types, the walker will recursively explore each element of the map continuing its search for
// strings to replace.
func (w *walker) hash(currentMap map[interface{}]interface{}, yamlPath []string) error {
	for configKey := range currentMap {
		path := yamlPath
		if newkey, ok := configKey.(string); ok {
			path = append(path, newkey)
		}

		switch v := currentMap[configKey].(type) {
		case string:
			if newValue, err := w.string(v, path); err == nil {
				currentMap[configKey] = newValue
			} else {
				return err
			}
		case map[interface{}]interface{}:
			if err := w.hash(v, path); err != nil {
				return err
			}
		case []interface{}:
			if err := w.slice(v, path); err != nil {
				return err
			}
		}
	}
	return nil
}

// walk recursively explores a loaded YAML in search for string values to replace. For each string the 'resolver' will
// be called allowing it to overwrite the string value.
func (w *walker) walk(data *interface{}) error {
	switch v := (*data).(type) {
	case map[interface{}]interface{}:
		return w.hash(v, nil)
	case []interface{}:
		return w.slice(v, nil)
	default:
		return fmt.Errorf("given data is not of expected type map not slice")
	}
}
