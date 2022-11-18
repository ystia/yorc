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
// It means either list, map or primitive type
func IsBuiltinType(typeName string) bool {
	// type representation for map and list could be map:<EntrySchema> or list:<EntrySchema> (ex: list:integer)
	return strings.HasPrefix(typeName, "list") || strings.HasPrefix(typeName, "map") || IsPrimitiveType(typeName)
}

// IsPrimitiveType checks if a given type name corresponds to a primitive type
// It means a data type that can'be broken down into a more simple data type.
// Known primitive types:
//   - string
//   - integer
//   - float
//   - boolean
//   - timestamp
//   - null
//   - version
//   - range
//   - scalar-unit.size
//   - scalar-unit.time
//   - scalar-unit.frequency
//   - scalar-unit.bitrate
func IsPrimitiveType(typeName string) bool {
	return typeName == "string" || typeName == "integer" || typeName == "float" || typeName == "boolean" ||
		typeName == "timestamp" || typeName == "null" || typeName == "version" || typeName == "range" ||
		typeName == "scalar-unit.size" || typeName == "scalar-unit.time" ||
		typeName == "scalar-unit.frequency" || typeName == "scalar-unit.bitrate"
}

//go:generate go-enum -f=types.go

// TypeBase is an enumerated type for TOSCA base types
/*
ENUM(
NODE
RELATIONSHIP
CAPABILITY
POLICY
ARTIFACT
DATA
)
*/
type TypeBase int

// Type is the base type for all TOSCA types (like node types, relationship types, ...)
type Type struct {
	Base        TypeBase          `yaml:"base,omitempty" json:"base,omitempty"`
	DerivedFrom string            `yaml:"derived_from,omitempty" json:"derived_from,omitempty"`
	Version     string            `yaml:"version,omitempty" json:"version,omitempty"`
	ImportPath  string            `yaml:"import_path,omitempty" json:"import_path,omitempty"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// An NodeType is the representation of a TOSCA Node Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_NODE_TYPE
// for more details
type NodeType struct {
	Type         `yaml:",inline"`
	Properties   map[string]PropertyDefinition   `yaml:"properties,omitempty" json:"properties,omitempty"`
	Attributes   map[string]AttributeDefinition  `yaml:"attributes,omitempty" json:"attributes,omitempty"`
	Requirements []RequirementDefinitionMap      `yaml:"requirements,omitempty,flow" json:"requirements,omitempty"`
	Capabilities map[string]CapabilityDefinition `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Interfaces   map[string]InterfaceDefinition  `yaml:"interfaces,omitempty" json:"interfaces,omitempty"`
	Artifacts    ArtifactDefMap                  `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
}

// An RelationshipType is the representation of a TOSCA Relationship Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_RELATIONSHIP_TYPE
// for more details
type RelationshipType struct {
	Type             `yaml:",inline"`
	Properties       map[string]PropertyDefinition  `yaml:"properties,omitempty" json:"properties,omitempty"`
	Attributes       map[string]AttributeDefinition `yaml:"attributes,omitempty" json:"attributes,omitempty"`
	Interfaces       map[string]InterfaceDefinition `yaml:"interfaces,omitempty" json:"interfaces,omitempty"`
	Artifacts        ArtifactDefMap                 `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	ValidTargetTypes []string                       `yaml:"valid_target_types,omitempty" json:"valid_target_types,omitempty"`
}

// An CapabilityType is the representation of a TOSCA Capability Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_CAPABILITY_TYPE
// for more details
type CapabilityType struct {
	Type             `yaml:",inline"`
	Properties       map[string]PropertyDefinition  `yaml:"properties,omitempty" json:"properties,omitempty"`
	Attributes       map[string]AttributeDefinition `yaml:"attributes,omitempty" json:"attributes,omitempty"`
	ValidSourceTypes []string                       `yaml:"valid_source_types,omitempty,flow" json:"valid_source_types,omitempty"`
}

// An ArtifactType is the representation of a TOSCA Artifact Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_ARTIFACT_TYPE
// for more details
type ArtifactType struct {
	Type       `yaml:",inline"`
	MimeType   string                        `yaml:"mime_type,omitempty" json:"mime_type,omitempty"`
	FileExt    []string                      `yaml:"file_ext,omitempty" json:"file_ext,omitempty"`
	Properties map[string]PropertyDefinition `yaml:"properties,omitempty" json:"properties,omitempty"`
}

// An DataType is the representation of a TOSCA Data Type
//
// See http://docs.oasis-open.org/tosca/TOSCA-Simple-Profile-YAML/v1.2/TOSCA-Simple-Profile-YAML-v1.2.html#DEFN_ENTITY_DATA_TYPE
// for more details
type DataType struct {
	Type       `yaml:",inline"`
	Properties map[string]PropertyDefinition `yaml:"properties,omitempty" json:"properties,omitempty"`
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
	Properties map[string]PropertyDefinition `yaml:"properties,omitempty" json:"properties,omitempty"`
	Targets    []string                      `yaml:"targets,omitempty,flow" json:"targets,omitempty"`
}
