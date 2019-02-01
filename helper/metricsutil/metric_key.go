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

package metricsutil

import (
	"strings"
)

// CleanupMetricKey replaces any reserved characters in statsd/statsite and prometheus by '-'.
func CleanupMetricKey(key []string) []string {
	res := make([]string, len(key))
	for i, keyPart := range key {
		// . is the statsd separator, _ is the prometheus separator, / is not allowed by prometheus | and : are reserved separators for statsd
		res[i] = strings.Replace(keyPart, "/", "-", -1)
		res[i] = strings.Replace(res[i], ".", "-", -1)
		res[i] = strings.Replace(res[i], "_", "-", -1)
		res[i] = strings.Replace(res[i], "|", "-", -1)
		res[i] = strings.Replace(res[i], ":", "-", -1)
		// " should be banned from keys
		res[i] = strings.Replace(res[i], "\"", "", -1)
	}
	return res
}
