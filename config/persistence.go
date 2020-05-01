package config

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"strings"

	"github.com/safing/portbase/log"
)

var (
	configFilePath string
)

func loadConfig() error {
	// check if persistence is configured
	if configFilePath == "" {
		return nil
	}

	// read config file
	data, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return err
	}

	// convert to map
	newValues, err := JSONToMap(data)
	if err != nil {
		return err
	}

	// apply
	return setConfig(newValues)
}

func saveConfig() error {
	// check if persistence is configured
	if configFilePath == "" {
		return nil
	}

	// extract values
	activeValues := make(map[string]interface{})
	optionsLock.RLock()
	for key, option := range options {
		option.Lock()
		if option.activeValue != nil {
			activeValues[key] = option.activeValue.getData(option)
		}
		option.Unlock()
	}
	optionsLock.RUnlock()

	// convert to JSON
	data, err := MapToJSON(activeValues)
	if err != nil {
		log.Errorf("config: failed to save config: %s", err)
		return err
	}

	// write file
	return ioutil.WriteFile(configFilePath, data, 0600)
}

// JSONToMap parses and flattens a hierarchical json object.
func JSONToMap(jsonData []byte) (map[string]interface{}, error) {
	loaded := make(map[string]interface{})
	err := json.Unmarshal(jsonData, &loaded)
	if err != nil {
		return nil, err
	}

	return Flatten(loaded), nil
}

// Flatten returns a flattened copy of the given hierarchical config.
func Flatten(config map[string]interface{}) (flattenedConfig map[string]interface{}) {
	flattenedConfig = make(map[string]interface{})
	flattenMap(flattenedConfig, config, "")
	return flattenedConfig
}

func flattenMap(rootMap, subMap map[string]interface{}, subKey string) {
	for key, entry := range subMap {

		// get next level key
		subbedKey := path.Join(subKey, key)

		// check for next subMap
		nextSub, ok := entry.(map[string]interface{})
		if ok {
			flattenMap(rootMap, nextSub, subbedKey)
		} else {
			// only set if not on root level
			rootMap[subbedKey] = entry
		}
	}
}

// MapToJSON expands a flattened map and returns it as json.
func MapToJSON(config map[string]interface{}) ([]byte, error) {
	return json.MarshalIndent(Expand(config), "", "  ")
}

// Expand returns a hierarchical copy of the given flattened config.
func Expand(flattenedConfig map[string]interface{}) (config map[string]interface{}) {
	config = make(map[string]interface{})
	for key, entry := range flattenedConfig {
		PutValueIntoHierarchicalConfig(config, key, entry)
	}
	return config
}

// PutValueIntoHierarchicalConfig injects a configuration entry into an hierarchical config map. Conflicting entries will be replaced.
func PutValueIntoHierarchicalConfig(config map[string]interface{}, key string, value interface{}) {
	parts := strings.Split(key, "/")

	// create/check maps for all parts except the last one
	subMap := config
	for i, part := range parts {
		if i == len(parts)-1 {
			// do not process the last part,
			// which is not a map, but the value key itself
			break
		}

		var nextSubMap map[string]interface{}
		// get value
		value, ok := subMap[part]
		if !ok {
			// create new map and assign it
			nextSubMap = make(map[string]interface{})
			subMap[part] = nextSubMap
		} else {
			nextSubMap, ok = value.(map[string]interface{})
			if !ok {
				// create new map and assign it
				nextSubMap = make(map[string]interface{})
				subMap[part] = nextSubMap
			}
		}

		// assign for next parts loop
		subMap = nextSubMap
	}

	// assign value to last submap
	subMap[parts[len(parts)-1]] = value
}

// CleanFlattenedConfig removes all inexistent configuration options from the given flattened config map.
func CleanFlattenedConfig(flattenedConfig map[string]interface{}) {
	optionsLock.RLock()
	defer optionsLock.RUnlock()

	for key := range flattenedConfig {
		_, ok := options[key]
		if !ok {
			delete(flattenedConfig, key)
		}
	}
}

// CleanHierarchicalConfig removes all inexistent configuration options from the given hierarchical config map.
func CleanHierarchicalConfig(config map[string]interface{}) {
	optionsLock.RLock()
	defer optionsLock.RUnlock()

	cleanSubMap(config, "")
}

func cleanSubMap(subMap map[string]interface{}, subKey string) (empty bool) {
	var foundValid int
	for key, value := range subMap {
		value, ok := value.(map[string]interface{})
		if ok {
			// we found another section
			isEmpty := cleanSubMap(value, path.Join(subKey, key))
			if isEmpty {
				delete(subMap, key)
			} else {
				foundValid++
			}
			continue
		}

		// we found an option value
		if strings.Contains(key, "/") {
			delete(subMap, key)
		} else {
			_, ok := options[path.Join(subKey, key)]
			if ok {
				foundValid++
			} else {
				delete(subMap, key)
			}
		}
	}
	return foundValid == 0
}
