package settings

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Load loads the given reader in settings
func Load(r io.Reader, ops ...func(*Options)) (Settings, error) {
	options := &Options{}
	for _, op := range ops {
		op(options)
	}

	s := make(map[interface{}]interface{})
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&s); err != nil {
		if err == io.EOF {
			return Settings{}, nil
		}
		return nil, errors.Wrap(err, "failed to read settings")
	}
	converted, err := convertToStringKeysRecursive(s, "")
	if err != nil {
		return nil, err
	}
	settings := converted.(map[string]interface{})
	if options.prefix != "" {
		settings = map[string]interface{}{
			options.prefix: settings,
		}
	}
	return settings, nil
}

// LoadFile loads a file (path) in settings (i.e. flatten map)
func LoadFile(path string, ops ...func(*Options)) (Settings, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return Load(r, ops...)
}

// LoadData loads data in settings
func LoadData(data []byte, ops ...func(*Options)) (Settings, error) {
	options := &Options{}
	for _, op := range ops {
		op(options)
	}
	s := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(data, s); err != nil {
		return nil, err
	}
	converted, err := convertToStringKeysRecursive(s, "")
	if err != nil {
		return nil, err
	}
	settings := converted.(map[string]interface{})
	if options.prefix != "" {
		settings = map[string]interface{}{
			options.prefix: settings,
		}
	}
	return settings, nil
}

// LoadFiles loads multiple path in settings, merging them.
func LoadFiles(paths []string, ops ...func(*Options)) (Settings, error) {
	m := Settings(map[string]interface{}{})
	for _, path := range paths {
		settings, err := LoadFile(path, ops...)
		if err != nil {
			return nil, err
		}
		m, err = Merge(m, settings)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

// from cli
func convertToStringKeysRecursive(value interface{}, keyPrefix string) (interface{}, error) {
	if mapping, ok := value.(map[interface{}]interface{}); ok {
		dict := make(map[string]interface{})
		for key, entry := range mapping {
			str, ok := key.(string)
			if !ok {
				return nil, formatInvalidKeyError(keyPrefix, key)
			}
			var newKeyPrefix string
			if keyPrefix == "" {
				newKeyPrefix = str
			} else {
				newKeyPrefix = fmt.Sprintf("%s.%s", keyPrefix, str)
			}
			convertedEntry, err := convertToStringKeysRecursive(entry, newKeyPrefix)
			if err != nil {
				return nil, err
			}
			dict[str] = convertedEntry
		}
		return dict, nil
	}
	if list, ok := value.([]interface{}); ok {
		var convertedList []interface{}
		for index, entry := range list {
			newKeyPrefix := fmt.Sprintf("%s[%d]", keyPrefix, index)
			convertedEntry, err := convertToStringKeysRecursive(entry, newKeyPrefix)
			if err != nil {
				return nil, err
			}
			convertedList = append(convertedList, convertedEntry)
		}
		return convertedList, nil
	}
	return value, nil
}

func formatInvalidKeyError(keyPrefix string, key interface{}) error {
	var location string
	if keyPrefix == "" {
		location = "at top level"
	} else {
		location = fmt.Sprintf("in %s", keyPrefix)
	}
	return errors.Errorf("Non-string key %s: %#v", location, key)
}
