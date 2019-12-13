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

package types

// Credential is a tosca.datatypes.Credential as defined in the specification https://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.3/csprd01/TOSCA-Simple-Profile-YAML-v1.3-csprd01.html#TYPE_TOSCA_DATA_CREDENTIAL
type Credential struct {
	Protocol  string            `mapstructure:"protocol" json:"protocol"`
	TokenType string            `mapstructure:"token_type" json:"token_type"` // default: password
	Token     string            `mapstructure:"token" json:"token"`
	Keys      map[string]string `mapstructure:"keys" json:"keys"`
	User      string            `mapstructure:"user" json:"user"`
}

// ProvisioningBastion is a representation of yorc.datatypes.ProvisioningBastion.
type ProvisioningBastion struct {
	Use         string     `mapstructure:"use" json:"use"`
	Host        string     `mapstructure:"host" json:"host"`
	Port        string     `mapstructure:"port" json:"port"`
	Credentials Credential `mapstructure:"credentials" json:"credentials"`
}
