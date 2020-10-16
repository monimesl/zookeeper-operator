/*
 * Copyright 2020 Skulup Ltd, Open Collaborators
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

import (
	"fmt"
	"github.com/skulup/operator-helper/configs"
	"gopkg.in/yaml.v2"
	"strconv"
)

func GetInt32(key string, keyValues map[string]string) *int32 {
	if v := GetString(key, keyValues); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			i32 := int32(i)
			return &i32
		}
		fmt.Printf("Expecting key=%s to be int but isn't. Value=%s", key, v)
	}
	return nil
}

func GetString(key string, keyValues map[string]string) string {
	return keyValues[key]
}

// CreateConfig create a config string from a key-value map updated with the yaml extras excluding exclusions
func CreateConfig(extras string, name string, keyValues map[string]string, exclusions ...string) (string, map[string]string) {
	isIncluded := func(needle string) bool {
		for _, ex := range exclusions {
			if needle == ex {
				return false
			}
		}
		return true
	}
	if extras != "" {
		extrasMap := map[string]string{}
		if err := yaml.Unmarshal([]byte(extras), extrasMap); err != nil {
			fmt.Println(fmt.Errorf("invalid %s data. reason: %s", name, err))
		}
		for k, v := range extrasMap {
			if !isIncluded(k) {
				configs.RequireRootLogger().Info(
					fmt.Sprintf("The key cannot be set directly to '%s'. Skipping...", name), "key", k)
				continue
			}
			keyValues[k] = v
		}
	}
	config := ""
	for k, v := range keyValues {
		if k == "" {
			configs.RequireRootLogger().Info(
				fmt.Sprintf("Invalid '%s' key", name), "key", k)
		} else if v != "" {
			// drop empty value config
			config += fmt.Sprintf("%s=%s\n", k, v)
			continue
		}
		delete(keyValues, k)
	}
	return config, keyValues
}
