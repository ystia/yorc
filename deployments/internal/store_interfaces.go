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

package internal

import (
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v3/helper/consulutil"
	"github.com/ystia/yorc/v3/tosca"
)

func storeOperationOutput(consulStore consulutil.ConsulStore, valueAssign *tosca.ValueAssignment, operationOutputPrefix string) error {
	f := valueAssign.GetFunction()
	opOutputFuncs := f.GetFunctionsByOperator(tosca.GetOperationOutputOperator)
	for _, oof := range opOutputFuncs {
		if len(oof.Operands) != 4 {
			return errors.Errorf("Invalid %q TOSCA function: %v", tosca.GetOperationOutputOperator, oof)
		}
		entityName := url.QueryEscape(oof.Operands[0].String())
		if entityName == "TARGET" || entityName == "SOURCE" {
			return errors.Errorf("Can't use SOURCE or TARGET keyword in a %q in node type context: %v", tosca.GetOperationOutputOperator, oof)
		}
		interfaceName := strings.ToLower(url.QueryEscape(oof.Operands[1].String()))
		operationName := strings.ToLower(url.QueryEscape(oof.Operands[2].String()))
		outputVariableName := url.QueryEscape(oof.Operands[3].String())
		consulStore.StoreConsulKeyAsString(path.Join(operationOutputPrefix, interfaceName, operationName, "outputs", entityName, outputVariableName, "expression"), oof.String())
	}
	return nil
}

func storeInputDefinition(consulStore consulutil.ConsulStore, inputPrefix, operationOutputPrefix string, inputDef tosca.Input) error {
	isValueAssignment := false
	isPropertyDefinition := false
	if inputDef.ValueAssign != nil {
		StoreValueAssignment(consulStore, inputPrefix+"/data", inputDef.ValueAssign)
		isValueAssignment = true
		if inputDef.ValueAssign.Type == tosca.ValueAssignmentFunction {
			err := storeOperationOutput(consulStore, inputDef.ValueAssign, operationOutputPrefix)
			if err != nil {
				return err
			}
		}
	}
	if inputDef.PropDef != nil {
		consulStore.StoreConsulKeyAsString(inputPrefix+"/type", inputDef.PropDef.Type)
		StoreValueAssignment(consulStore, inputPrefix+"/default", inputDef.PropDef.Default)
		consulStore.StoreConsulKeyAsString(inputPrefix+"/status", inputDef.PropDef.Status)
		if inputDef.PropDef.Required == nil {
			// Required by default
			consulStore.StoreConsulKeyAsString(inputPrefix+"/required", "true")
		} else {
			consulStore.StoreConsulKeyAsString(inputPrefix+"/required", strconv.FormatBool(*inputDef.PropDef.Required))
		}
		isPropertyDefinition = true
	}

	consulStore.StoreConsulKeyAsString(inputPrefix+"/is_value_assignment", strconv.FormatBool(isValueAssignment))
	consulStore.StoreConsulKeyAsString(inputPrefix+"/is_property_definition", strconv.FormatBool(isPropertyDefinition))
	return nil
}

func storeInterfaces(consulStore consulutil.ConsulStore, interfaces map[string]tosca.InterfaceDefinition, prefix string, isRelationshipType bool) error {
	for intTypeName, intMap := range interfaces {
		intTypeName = strings.ToLower(intTypeName)
		interfacePrefix := path.Join(prefix, "interfaces", intTypeName)
		// Store Global inputs
		for inputName, inputDef := range intMap.Inputs {
			inputPrefix := path.Join(interfacePrefix, "inputs", inputName)
			err := storeInputDefinition(consulStore, inputPrefix, interfacePrefix, inputDef)
			if err != nil {
				return err
			}
		}

		for opName, operationDef := range intMap.Operations {
			opName = strings.ToLower(opName)
			operationPrefix := path.Join(interfacePrefix, opName)

			for inputName, inputDef := range operationDef.Inputs {
				inputPrefix := path.Join(operationPrefix, "inputs", inputName)
				err := storeInputDefinition(consulStore, inputPrefix, interfacePrefix, inputDef)
				if err != nil {
					return err
				}
			}
			if operationDef.Implementation.Artifact != (tosca.ArtifactDefinition{}) {
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/file", operationDef.Implementation.Artifact.File)
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/type", operationDef.Implementation.Artifact.Type)
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/repository", operationDef.Implementation.Artifact.Repository)
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/deploy_path", operationDef.Implementation.Artifact.DeployPath)

			} else {
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/primary", operationDef.Implementation.Primary)
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/dependencies", strings.Join(operationDef.Implementation.Dependencies, ","))
			}
			if operationDef.Implementation.OperationHost != "" {
				if err := checkOperationHost(operationDef.Implementation.OperationHost, isRelationshipType); err != nil {
					return err
				}
				consulStore.StoreConsulKeyAsString(operationPrefix+"/implementation/operation_host", strings.ToUpper(operationDef.Implementation.OperationHost))
			}
		}
	}
	return nil
}

func checkOperationHost(operationHost string, isRelationshipType bool) error {
	operationHostUpper := strings.ToUpper(operationHost)
	if isRelationshipType {
		// Allowed values are SOURCE, TARGET and ORCHESTRATOR
		switch operationHostUpper {
		case "ORCHESTRATOR":
		case "SOURCE":
		case "TARGET":
			return nil
		default:
			return errors.Errorf("Invalid value for Implementation operation_host property associated to a relationship type :%q", operationHost)

		}
	} else {
		// Allowed values are SELF, HOST and ORCHESTRATOR
		switch operationHostUpper {
		case "HOST":
		case "ORCHESTRATOR":
		case "SELF":
			return nil
		default:
			return errors.Errorf("Invalid value for Implementation operation_host property associated to non-relationship type:%q", operationHost)

		}
	}
	return nil
}
