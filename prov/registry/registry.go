package registry

import (
	"regexp"

	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

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
	ListDelegateExecutors() map[string]string
}

var defaultReg Registry

func init() {
	defaultReg = &defaultRegistry{delegateMatches: make([]delgateMatch, 0)}
}

func GetRegistry() Registry {
	return defaultReg
}

type delgateMatch struct {
	match    string
	executor prov.DelegateExecutor
	origin   string
}

type defaultRegistry struct {
	delegateMatches []delgateMatch
}

func (r *defaultRegistry) RegisterDelegates(matches []string, executor prov.DelegateExecutor, origin string) {
	for _, match := range matches {
		r.delegateMatches = append(r.delegateMatches, delgateMatch{match: match, executor: executor, origin: origin})
	}
}

func (r *defaultRegistry) GetDelegateExecutor(nodeType string) (prov.DelegateExecutor, error) {
	for _, m := range r.delegateMatches {
		ok, err := regexp.MatchString(m.match, nodeType)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to match delegate executor from nodeType %q", nodeType)
		}
		if ok {
			return m.executor, nil
		}
	}
	return nil, errors.Errorf("Unsupported node type %q for a delegate operation", nodeType)
}

func (r *defaultRegistry) ListDelegateExecutors() map[string]string {
	result := make(map[string]string, len(r.delegateMatches))
	for _, match := range r.delegateMatches {
		result[match.match] = match.origin
	}
	return result
}
