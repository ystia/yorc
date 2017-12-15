package registry

import (
	"regexp"
	"sync"

	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/prov"
	"novaforge.bull.com/starlings-janus/janus/vault"
)

// BuiltinOrigin is the origin for Janus builtin
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

	// Register a TOSCA definition file. Origin is the origin of the executor (builtin for builtin executors or the plugin name in case of a plugin)
	AddToscaDefinition(name, origin string, data []byte)
	// GetToscaDefinition retruns the definitions for the given name.
	//
	// If the given definition name can't match any definition an error is returned
	GetToscaDefinition(name string) ([]byte, error)
	// ListToscaDefinitions returns a map of definitions names to their origin
	ListToscaDefinitions() []Definition

	// RegisterDelegates register a list of implementation artifact type that should be used along with the given
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

// Definition represents a TOSCA definition with its Name, Origin and Data content
type Definition struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
	Data   []byte `json:"-"`
}

// VaultClientBuilder represents a vault client builder with its ID, Origin and Data content
type VaultClientBuilder struct {
	ID      string              `json:"id"`
	Origin  string              `json:"origin"`
	Builder vault.ClientBuilder `json:"-"`
}

type defaultRegistry struct {
	delegateMatches     []DelegateMatch
	operationMatches    []OperationExecMatch
	definitions         []Definition
	vaultClientBuilders []VaultClientBuilder
	delegatesLock       sync.RWMutex
	operationsLock      sync.RWMutex
	definitionsLock     sync.RWMutex
	vaultsLock          sync.RWMutex
}

func (r *defaultRegistry) RegisterDelegates(matches []string, executor prov.DelegateExecutor, origin string) {
	r.delegatesLock.Lock()
	defer r.delegatesLock.Unlock()
	if len(matches) > 0 {
		newDelegates := make([]DelegateMatch, len(matches))
		for i := range matches {
			newDelegates[i] = DelegateMatch{Match: matches[i], Executor: executor, Origin: origin}
		}
		// Put them at the begining
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

func (r *defaultRegistry) AddToscaDefinition(name, origin string, data []byte) {
	r.definitionsLock.Lock()
	defer r.definitionsLock.Unlock()
	// Insert as first
	r.definitions = append([]Definition{Definition{name, origin, data}}, r.definitions...)
}

func (r *defaultRegistry) GetToscaDefinition(name string) ([]byte, error) {
	r.definitionsLock.RLock()
	defer r.definitionsLock.RUnlock()
	for _, def := range r.definitions {
		if def.Name == name {
			return def.Data, nil
		}
	}
	return nil, errors.Errorf("Unknown definition: %q", name)
}

func (r *defaultRegistry) ListToscaDefinitions() []Definition {
	r.definitionsLock.RLock()
	defer r.definitionsLock.RUnlock()
	result := make([]Definition, len(r.definitions))
	copy(result, r.definitions)
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
		// Put them at the begining
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
