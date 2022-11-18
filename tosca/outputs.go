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

package tosca

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/log"
	"strings"
)

// An Output is the representation of the output part of a TOSCA Operation Definition
type Output struct {
	ValueAssign      *ValueAssignment  `json:"value_assignment,omitempty"`
	AttributeMapping *AttributeMapping `json:"attribute_mapping,omitempty"`
}

// AttributeMapping is the representation of a Tosca attribute mapping
// Attribute mappings have the following grammar :
// [ <SELF | SOURCE | TARGET >, <optional_capability_name>, <attribute_name>, <nested_attribute_name_or_index_1>, ..., <nested_attribute_name_or_index_or_key_n> ]
type AttributeMapping struct {
	Parameters []string
}

func (am *AttributeMapping) String() string {
	return fmt.Sprintf("Parameters:%+v", am.Parameters)
}

// Examples: [ SELF, endpoint, ip_address ] , [ SELF, endpoint, credentials, user ]
func parseAttributeMapping(attributeMappingSeq []string) (*AttributeMapping, error) {
	log.Debugf("Parsing attribute mapping sequence:%v", attributeMappingSeq)

	// At least 2 parameters are required
	if len(attributeMappingSeq) < 2 {
		return nil, errors.Errorf("Expecting at least 2 parameters for the attribute mapping sequence:%q", attributeMappingSeq)
	}

	// Check first parameter is SELF, SOURCE or TARGET
	param0 := strings.ToUpper(attributeMappingSeq[0])
	if param0 != Self && param0 != Source && param0 != Target {
		return nil, errors.Errorf("Expecting first parameter being SELF, SOURCE or TARGET for the attribute mapping sequence:%q", attributeMappingSeq)
	}

	attributeMapping := AttributeMapping{Parameters: attributeMappingSeq}
	for i := range attributeMapping.Parameters {
		attributeMapping.Parameters[i] = strings.TrimSpace(attributeMapping.Parameters[i])
	}

	return &attributeMapping, nil
}
