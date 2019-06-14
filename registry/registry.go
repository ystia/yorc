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

package registry

import (
	"regexp"
	"sync"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/vault"
)

// BuiltinOrigin is the origin for Yorc builtin
const BuiltinOrigin = "builtin"

// Registry holds references to for node types matches to prov.DelegateExecutor
type Registry interface {
	// RegisterDelegates register a list of node type regexp matches that should be used along with the given
	// prov.DelegateExecutor. Origin is the origin of the executor (builtin for builtin executors or the plugin name in case of a plugin)
	RegisterDelegates(matches []string, executor prov.DelegateExecutor, origin string)
	// Returns the first prov.DelegateExecutor that matches the given nodeType
	//
	// If the given nodeType can't match any prov.DelegateExecutor an error is returned
	GetDelegateExecutor(nodeType string) (prov.DelegateExecutor, error)
	// ListDelegateExecutors returns a map of node types matches to prov.DelegateExecutor origin
	ListDelegateExecutors() []DelegateMatch

	// RegisterOperationExecutor register a list of implementation artifact type that should be used along with the given
	// prov.OperationExecutor. Origin is the origin of the executor (builtin for builtin executors or the plugin name in case of a plugin)
	RegisterOperationExecutor(artifacts []string, executor prov.OperationExecutor, origin string)
	// Returns the first prov.OperationExecutor that matches the given artifact implementation
	//
	// If the given nodeType can't match any prov.DelegateExecutor an error is returned
	GetOperationExecutor(artifact string) (prov.OperationExecutor, error)
	// ListOperationExecutors returns a map of node types matches to prov.DelegateExecutor origin
	ListOperationExecutors() []OperationExecMatch

	// Register a Vault Client Builder. Origin is the origin of the client builder (builtin for builtin vault client builder or the plugin name in case of a plugin)
	RegisterVaultClientBuilder(id string, builder vault.ClientBuilder, origin string)
	// Returns the first vault.ClientBuilder that matches the given id
	//
	// If the given id can't match any vault.ClientBuilder an error is returned
	GetVaultClientBuilder(id string) (vault.ClientBuilder, error)

	// ListVaultClients returns a list of registered Vault Clients origin
	ListVaultClientBuilders() []VaultClientBuilder

	// RegisterInfraUsageCollector allows to register an infrastructure usage collector with its name as unique id
	RegisterInfraUsageCollector(name string, infraUsageCollector prov.InfraUsageCollector, origin string)

	// GetInfraUsageCollector returns a prov.Infrastructure from its name as unique id
	//
	// If the given id can't match any prov.Infrastructure an error is returned
	GetInfraUsageCollector(name string) (prov.InfraUsageCollector, error)

	// ListInfraUsageCollectors returns a list of registered infrastructure usage collectors origin
	ListInfraUsageCollectors() []InfraUsageCollector

	// RegisterActionOperator register a list of actionTypes that should be used along with the given
	// prov.ActionOperator. Origin is the origin of the operator (builtin for builtin operators or the plugin name in case of a plugin)
	RegisterActionOperator(actionTypes []string, operator prov.ActionOperator, origin string)
	// Returns the first prov.ActionOperator that matches the given actionType
	//
	// If the given actionType can't match any prov.ActionOperator, an error is returned
	GetActionOperator(actionType string) (prov.ActionOperator, error)
	// ListActionOperators returns a map of actionTypes matches to prov.ActionOperator origin
	ListActionOperators() []ActionTypeMatch

	// Deprecated

	// Register a TOSCA definition file. Origin is the origin of the executor (builtin for builtin executors or the plugin name in case of a plugin)
	//
	// Deprecated: use store.CommonDefinition instead
	//             will be removed in Yorc 4.0
	AddToscaDefinition(name, origin string, data []byte)
	// GetToscaDefinition retruns the definitions for the given name.
	//
	// If the given definition name can't match any definition an error is returned
	//
	// Deprecated: use store.GetCommonsTypesPaths instead to know supported definitions
	//             will be removed in Yorc 4.0
	GetToscaDefinition(name string) ([]byte, error)
	// ListToscaDefinitions returns a map of definitions names to their origin
	//
	// Deprecated: use store.GetCommonsTypesPaths instead to know supported definitions
	//             will be removed in Yorc 4.0
	ListToscaDefinitions() []Definition
}

var defaultReg Registry

func init() {
	defaultReg = &defaultRegistry{delegateMatches: make([]DelegateMatch, 0), definitions: make([]Definition, 0), vaultClientBuilders: make([]VaultClientBuilder, 0)}
}

// GetRegistry returns the singleton instance of the Registry
func GetRegistry() Registry {
	return defaultReg
}

// DelegateMatch represents a matching between a node type regexp match and a DelegateExecutor from a given origin
type DelegateMatch struct {
	Match    string                `json:"node_type"`
	Executor prov.DelegateExecutor `json:"-"`
	Origin   string                `json:"origin"`
}

// OperationExecMatch represents a matching between an implementation artifact match and an OperationExecutor from a given origin
type OperationExecMatch struct {
	Artifact string                 `json:"implementation_artifact"`
	Executor prov.OperationExecutor `json:"-"`
	Origin   string                 `json:"origin"`
}

// ActionTypeMatch represents a matching between an actionType and an ActionOperator from a given origin
type ActionTypeMatch struct {
	ActionType string              `json:"action_type"`
	Operator   prov.ActionOperator `json:"-"`
	Origin     string              `json:"origin"`
}

// VaultClientBuilder represents a vault client builder with its ID, Origin and Data content
type VaultClientBuilder struct {
	ID      string              `json:"id"`
	Origin  string              `json:"origin"`
	Builder vault.ClientBuilder `json:"-"`
}

// InfraUsageCollector represents an infrastructure usage collector with its Name, Origin and Data content
type InfraUsageCollector struct {
	Name                string                   `json:"id"`
	Origin              string                   `json:"origin"`
	InfraUsageCollector prov.InfraUsageCollector `json:"-"`
}

type defaultRegistry struct {
	delegateMatches          []DelegateMatch
	operationMatches         []OperationExecMatch
	actionTypeMatches        []ActionTypeMatch
	definitions              []Definition
	vaultClientBuilders      []VaultClientBuilder
	infraUsageCollectors     []InfraUsageCollector
	delegatesLock            sync.RWMutex
	operationsLock           sync.RWMutex
	definitionsLock          sync.RWMutex
	vaultsLock               sync.RWMutex
	infraUsageCollectorsLock sync.RWMutex
	actionOperatorsLock      sync.RWMutex
}

func (r *defaultRegistry) RegisterDelegates(matches []string, executor prov.DelegateExecutor, origin string) {
	r.delegatesLock.Lock()
	defer r.delegatesLock.Unlock()
	if len(matches) > 0 {
		newDelegates := make([]DelegateMatch, len(matches))
		for i := range matches {
			newDelegates[i] = DelegateMatch{Match: matches[i], Executor: executor, Origin: origin}
		}
		// Put them at the beginning
		r.delegateMatches = append(newDelegates, r.delegateMatches...)
	}
}

func (r *defaultRegistry) GetDelegateExecutor(nodeType string) (prov.DelegateExecutor, error) {
	r.delegatesLock.RLock()
	defer r.delegatesLock.RUnlock()
	for _, m := range r.delegateMatches {
		ok, err := regexp.MatchString(m.Match, nodeType)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to match delegate executor from nodeType %q", nodeType)
		}
		if ok {
			return m.Executor, nil
		}
	}
	return nil, errors.Errorf("Unsupported node type %q for a delegate operation", nodeType)
}

func (r *defaultRegistry) ListDelegateExecutors() []DelegateMatch {
	r.delegatesLock.RLock()
	defer r.delegatesLock.RUnlock()
	result := make([]DelegateMatch, len(r.delegateMatches))
	copy(result, r.delegateMatches)
	return result
}

func (r *defaultRegistry) RegisterOperationExecutor(matches []string, executor prov.OperationExecutor, origin string) {
	r.operationsLock.Lock()
	defer r.operationsLock.Unlock()
	if len(matches) > 0 {
		newOpExecMatch := make([]OperationExecMatch, len(matches))
		for i := range matches {
			newOpExecMatch[i] = OperationExecMatch{Artifact: matches[i], Executor: executor, Origin: origin}
		}
		// Put them at the beginning
		r.operationMatches = append(newOpExecMatch, r.operationMatches...)
	}
}

func (r *defaultRegistry) GetOperationExecutor(artifact string) (prov.OperationExecutor, error) {
	r.operationsLock.RLock()
	defer r.operationsLock.RUnlock()
	for _, m := range r.operationMatches {
		if artifact == m.Artifact {
			return m.Executor, nil
		}
	}
	return nil, errors.Errorf("Unsupported artifact implementation %q for a call-operation", artifact)
}

func (r *defaultRegistry) ListOperationExecutors() []OperationExecMatch {
	r.operationsLock.RLock()
	defer r.operationsLock.RUnlock()
	result := make([]OperationExecMatch, len(r.operationMatches))
	copy(result, r.operationMatches)
	return result
}

func (r *defaultRegistry) RegisterVaultClientBuilder(id string, builder vault.ClientBuilder, origin string) {
	r.vaultsLock.Lock()
	defer r.vaultsLock.Unlock()
	// Insert as first
	r.vaultClientBuilders = append([]VaultClientBuilder{VaultClientBuilder{ID: id, Origin: origin, Builder: builder}}, r.vaultClientBuilders...)
}
func (r *defaultRegistry) GetVaultClientBuilder(id string) (vault.ClientBuilder, error) {
	r.vaultsLock.RLock()
	defer r.vaultsLock.RUnlock()
	for _, v := range r.vaultClientBuilders {
		if v.ID == id {
			return v.Builder, nil
		}
	}
	return nil, errors.Errorf("Unknown vault id: %q", id)
}

func (r *defaultRegistry) ListVaultClientBuilders() []VaultClientBuilder {
	r.vaultsLock.RLock()
	defer r.vaultsLock.RUnlock()
	result := make([]VaultClientBuilder, len(r.vaultClientBuilders))
	copy(result, r.vaultClientBuilders)
	return result
}

func (r *defaultRegistry) RegisterInfraUsageCollector(name string, infraUsageCollector prov.InfraUsageCollector, origin string) {
	r.infraUsageCollectorsLock.RLock()
	defer r.infraUsageCollectorsLock.RUnlock()
	r.infraUsageCollectors = append([]InfraUsageCollector{{Name: name, Origin: origin, InfraUsageCollector: infraUsageCollector}})
}

func (r *defaultRegistry) GetInfraUsageCollector(name string) (prov.InfraUsageCollector, error) {
	r.infraUsageCollectorsLock.RLock()
	defer r.infraUsageCollectorsLock.RUnlock()
	for _, pr := range r.infraUsageCollectors {
		if pr.Name == name {
			return pr.InfraUsageCollector, nil
		}
	}
	return nil, errors.Errorf("Unknown infra usage collector with name: %q", name)
}

func (r *defaultRegistry) ListInfraUsageCollectors() []InfraUsageCollector {
	r.infraUsageCollectorsLock.RLock()
	defer r.infraUsageCollectorsLock.RUnlock()
	result := make([]InfraUsageCollector, len(r.infraUsageCollectors))
	copy(result, r.infraUsageCollectors)
	return result
}

func (r *defaultRegistry) RegisterActionOperator(actionTypes []string, operator prov.ActionOperator, origin string) {
	r.actionOperatorsLock.Lock()
	defer r.actionOperatorsLock.Unlock()
	if len(actionTypes) > 0 {
		newActionOpMatch := make([]ActionTypeMatch, len(actionTypes))
		for i := range actionTypes {
			newActionOpMatch[i] = ActionTypeMatch{ActionType: actionTypes[i], Operator: operator, Origin: origin}
		}
		// Put them at the beginning
		r.actionTypeMatches = append(newActionOpMatch, r.actionTypeMatches...)
	}
}

func (r *defaultRegistry) GetActionOperator(actionType string) (prov.ActionOperator, error) {
	r.actionOperatorsLock.RLock()
	defer r.actionOperatorsLock.RUnlock()
	for _, m := range r.actionTypeMatches {
		if actionType == m.ActionType {
			return m.Operator, nil
		}
	}
	return nil, errors.Errorf("Unsupported actionType:%q for any registered action operators", actionType)
}

func (r *defaultRegistry) ListActionOperators() []ActionTypeMatch {
	r.actionOperatorsLock.RLock()
	defer r.actionOperatorsLock.RUnlock()
	result := make([]ActionTypeMatch, len(r.actionTypeMatches))
	copy(result, r.actionTypeMatches)
	return result
}
