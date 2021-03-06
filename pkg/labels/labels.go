/*
Copyright 2014 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package labels

import (
	"sort"
	"strings"
)

// Labels allows you to present labels independently from their storage.
type Labels interface {
	// Get returns the value for the provided label.
	Get(label string) (value string)
}

// Set is a map of label:value. It implements Labels.
type Set map[string]string

// String returns all labels listed as a human readable string.
// Conveniently, exactly the format that ParseSelector takes.
func (ls Set) String() string {
	selector := make([]string, 0, len(ls))
	for key, value := range ls {
		selector = append(selector, key+"="+value)
	}
	// Sort for determinism.
	sort.StringSlice(selector).Sort()
	return strings.Join(selector, ",")
}

// Get returns the value in the map for the provided label.
func (ls Set) Get(label string) string {
	return ls[label]
}

// AsSelector converts labels into a selectors.
func (ls Set) AsSelector() Selector {
	return SelectorFromSet(ls)
}
