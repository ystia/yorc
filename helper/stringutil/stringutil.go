// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stringutil

import (
	"strconv"
	"strings"
	"time"
)

// GetLastElement returns the last element of a "separator-separated" string
func GetLastElement(str string, separator string) string {
	var ret string
	if strings.Contains(str, separator) {
		idx := strings.LastIndex(str, separator)
		ret = str[idx+1:]
	}
	return ret
}

// GetAllExceptLastElement returns all elements except the last one of a "separator-separated" string
func GetAllExceptLastElement(str string, separator string) string {
	var ret string
	if strings.Contains(str, separator) {
		idx := strings.LastIndex(str, separator)
		ret = str[:idx]
	}
	return ret
}

// UniqueTimestampedName generates a time-stamped name for temporary file or directory by instance
func UniqueTimestampedName(prefix string, suffix string) string {
	return prefix + strconv.FormatInt(time.Now().UnixNano(), 10) + suffix
}
