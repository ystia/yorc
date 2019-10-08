// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package hostspool

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUpdateHost(t *testing.T) {
	err := updateHost(&httpClientMockDelete{}, []string{"hostOne"}, "locationOne", "", "", "pass", "userOne", "1.2.3.1", 22, []string{"label1=value1", "label2=value2", "label3=value3"}, []string{"label4=value4"})
	require.NoError(t, err, "Failed to add host")
}

func TestUpdateHostWithoutHostname(t *testing.T) {
	err := updateHost(&httpClientMockDelete{}, []string{}, "locationOne", "", "", "pass", "userOne", "1.2.3.1", 22, []string{"label1=value1", "label2=value2", "label3=value3"}, []string{"label4=value4"})
	require.Error(t, err, "Expected error as no hostname has been provided")
}

func TestUpdateHostWithoutLocation(t *testing.T) {
	err := updateHost(&httpClientMockDelete{}, []string{"hostOne"}, "", "", "", "pass", "userOne", "1.2.3.1", 22, []string{"label1=value1", "label2=value2", "label3=value3"}, []string{"label4=value4"})
	require.Error(t, err, "Expected error as no location has been provided")
}

func TestUpdateHostWithHTTPFailure(t *testing.T) {
	err := updateHost(&httpClientMockDelete{testID: "fails"}, []string{}, "locationOne", "", "", "pass", "userOne", "1.2.3.1", 22, []string{"label1=value1", "label2=value2", "label3=value3"}, []string{"label4=value4"})
	require.Error(t, err, "Expected error due to HTTP failure")
}

func TestUpdateHostWithJSONError(t *testing.T) {
	err := updateHost(&httpClientMockDelete{testID: "bad_json"}, []string{}, "locationOne", "", "", "pass", "userOne", "1.2.3.1", 22, []string{"label1=value1", "label2=value2", "label3=value3"}, []string{"label4=value4"})
	require.Error(t, err, "Expected error due to JSON error")
}
