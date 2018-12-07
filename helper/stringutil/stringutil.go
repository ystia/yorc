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
	"github.com/ystia/yorc/log"
	"io/ioutil"
	"os"
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

// GetFilePath takes in argument a string which can be a path to a file or the
// content of a file (some config values can be either a path or a content)
// If the argument is a file path, this function returns immediately this path value,
// and a boolean set to true meaning the argument was a path.
// If the argument is not a path, a new file will be created, the argument will be
// written in this file, the function will return the path to this new file,
// and a boolean value set to false meaning the argument wasn't a path
func GetFilePath(pathOrContent string) (string, bool, error) {
	if _, err := os.Stat(pathOrContent); err == nil {
		// This is a path
		return pathOrContent, true, err
	}

	// Create a file and write the argument in this file
	file, err := ioutil.TempFile("", "yorc")
	if err != nil {
		return "", false, err
	}

	filepath := file.Name()
	if _, err := file.WriteString(pathOrContent); err != nil {
		file.Close()
		return "", false, err
	}

	file.Sync()

	return filepath, false, file.Close()
}

// Truncate truncates a string if longer than defined max length
// The defined max length must be > 3 otherwise no truncation will be done.
// The returned string length is "max length" and ends with "..."
func Truncate(str string, maxLength int) string {
	if maxLength < 3 {
		log.Print("maxLength parameter must be greater than 3. No truncation will be done")
		return str
	}
	if len(str) > maxLength {
		return str[0:maxLength-3] + "..."
	}
	return str
}
