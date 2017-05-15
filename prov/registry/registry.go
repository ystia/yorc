package registry

import (
	"regexp"

	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/prov"
)

// Registry holds
type Registry interface {
	RegisterDelegates(matches []string, executor prov.DelegateExecutor)
	GetDelegateExecutor(nodeType string) (prov.DelegateExecutor, error)
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
}

type defaultRegistry struct {
	delegateMatches []delgateMatch
}

func (r *defaultRegistry) RegisterDelegates(matches []string, executor prov.DelegateExecutor) {
	for _, match := range matches {
		r.delegateMatches = append(r.delegateMatches, delgateMatch{match: match, executor: executor})
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
