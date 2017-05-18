package registry

import (
	"regexp"
	"sync"

	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/prov"
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
}

var defaultReg Registry

func init() {
	defaultReg = &defaultRegistry{delegateMatches: make([]DelegateMatch, 0), definitions: make([]Definition, 0)}
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

// Definition represents a TOSCA definition with its Name, Origin and Data content
type Definition struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
	Data   []byte `json:"-"`
}

type defaultRegistry struct {
	delegateMatches []DelegateMatch
	definitions     []Definition
	delegatesLock   sync.RWMutex
	definitionsLock sync.RWMutex
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
