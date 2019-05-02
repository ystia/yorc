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
	"strings"
)

// IsBuiltinType checks if a given type name corresponds to a TOSCA builtin type.
//
// Known builtin types:
// 	- string
//	- integer
//	- float
//	- boolean
//	- timestamp
//	- null
//	- list
//	- map
//	- version
//	- range
//	- scalar-unit.size
//	- scalar-unit.time
func IsBuiltinType(typeName string) bool {
	// type representation for map and list could be map:<EntrySchema> or list:<EntrySchema> (ex: list:integer)
	return strings.HasPrefix(typeName, "list") || strings.HasPrefix(typeName, "map") ||
		typeName == "string" || typeName == "integer" || typeName == "float" || typeName == "boolean" ||
		typeName == "timestamp" || typeName == "null" || typeName == "version" || typeName == "range" ||
		typeName == "scalar-unit.size" || typeName == "scalar-unit.time"
}

// Type is the base type for all TOSCA types (like node types, relationship types, ...)
type Type struct {
	DerivedFrom string            `yaml:"derived_from,omitempty"`
	Version     string            `yaml:"version,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
}

// An NodeType is the representation of a TOSCA Node Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_NODE_TYPE
// for more details
type NodeType struct {
	Type         `yaml:",inline"`
	Properties   map[string]PropertyDefinition   `yaml:"properties,omitempty"`
	Attributes   map[string]AttributeDefinition  `yaml:"attributes,omitempty"`
	Requirements []RequirementDefinitionMap      `yaml:"requirements,omitempty,flow"`
	Capabilities map[string]CapabilityDefinition `yaml:"capabilities,omitempty"`
	Interfaces   map[string]InterfaceDefinition  `yaml:"interfaces,omitempty"`
	Artifacts    ArtifactDefMap                  `yaml:"artifacts,omitempty"`
}

// An RelationshipType is the representation of a TOSCA Relationship Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_RELATIONSHIP_TYPE
// for more details
type RelationshipType struct {
	Type             `yaml:",inline"`
	Properties       map[string]PropertyDefinition  `yaml:"properties,omitempty"`
	Attributes       map[string]AttributeDefinition `yaml:"attributes,omitempty"`
	Interfaces       map[string]InterfaceDefinition `yaml:"interfaces,omitempty"`
	Artifacts        ArtifactDefMap                 `yaml:"artifacts,omitempty"`
	ValidTargetTypes []string                       `yaml:"valid_target_types,omitempty"`
}

// An CapabilityType is the representation of a TOSCA Capability Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_CAPABILITY_TYPE
// for more details
type CapabilityType struct {
	Type             `yaml:",inline"`
	Properties       map[string]PropertyDefinition  `yaml:"properties,omitempty"`
	Attributes       map[string]AttributeDefinition `yaml:"attributes,omitempty"`
	ValidSourceTypes []string                       `yaml:"valid_source_types,omitempty,flow"`
}

// An ArtifactType is the representation of a TOSCA Artifact Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_ARTIFACT_TYPE
// for more details
type ArtifactType struct {
	Type       `yaml:",inline"`
	MimeType   string                        `yaml:"mime_type,omitempty"`
	FileExt    []string                      `yaml:"file_ext,omitempty"`
	Properties map[string]PropertyDefinition `yaml:"properties,omitempty"`
}

// An DataType is the representation of a TOSCA Data Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_DATA_TYPE
// for more details
type DataType struct {
	Type       `yaml:",inline"`
	Properties map[string]PropertyDefinition `yaml:"properties,omitempty"`
	// Constraints not enforced in Yorc so we don't parse them
	// Constraints []ConstraintClause
}

// A PolicyType is the representation of a TOSCA Policy Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_POLICY_TYPE
// Triggers are not supported
// for more details
type PolicyType struct {
	Type       `yaml:",inline"`
	Properties map[string]PropertyDefinition `yaml:"properties,omitempty"`
	Targets    []string                      `yaml:"targets,omitempty,flow"`
}
