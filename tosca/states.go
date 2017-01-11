package tosca

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// NodeState represent the state of a node instance this part is normative
//
// We added a non-normative special step "deleted" in order to track deleted instances
type NodeState int

const (
	// NodeStateInitial is a non-transitional state indicating that the node is not yet created.  Node only exists as a template definition.
	NodeStateInitial NodeState = iota
	// NodeStateCreating is a transitional state indicating that the Node is transitioning from initial state to created state.
	NodeStateCreating
	// NodeStateCreated is a non-transitional state indicating that Node software has been installed.
	NodeStateCreated
	// NodeStateConfiguring is a transitional state indicating that Node is transitioning from created state to configured state.
	NodeStateConfiguring
	// NodeStateConfigured is a non-transitional state indicating that Node has been configured prior to being started.
	NodeStateConfigured
	// NodeStateStarting is a transitional state indicating that Node is transitioning from configured state to started state.
	NodeStateStarting
	// NodeStateStarted is a non-transitional state indicating that Node is started.
	NodeStateStarted
	// NodeStateStopping is a transitional state indicating that Node is transitioning from its current state to a configured state.
	NodeStateStopping
	// NodeStateDeleting is a transitional state indicating that Node is transitioning from its current state to one where it is deleted.
	//
	// We diverge here from the specification that states "and its state is no longer tracked by the instance model".
	NodeStateDeleting
	// NodeStateError is a non-transitional state indicating that the Node is in an error state.
	NodeStateError
	// NodeStateDeleted is a non-transitional state indicating that the Node is deleted.
	NodeStateDeleted
)

const _NodeState_name = "initialcreatingcreatedconfiguringconfiguredstartingstartedstoppingdeletingerrordeleted"

var _NodeState_index = [...]uint8{0, 7, 15, 22, 33, 43, 51, 58, 66, 74, 79, 84}

func (i NodeState) String() string {
	if i < 0 || i >= NodeState(len(_NodeState_index)-1) {
		return fmt.Sprintf("State(%d)", i)
	}
	return _NodeState_name[_NodeState_index[i]:_NodeState_index[i+1]]
}

var _NodeStateNameToValue_map = map[string]NodeState{
	_NodeState_name[0:7]:   0,
	_NodeState_name[7:15]:  1,
	_NodeState_name[15:22]: 2,
	_NodeState_name[22:33]: 3,
	_NodeState_name[33:43]: 4,
	_NodeState_name[43:51]: 5,
	_NodeState_name[51:58]: 6,
	_NodeState_name[58:66]: 7,
	_NodeState_name[66:74]: 8,
	_NodeState_name[74:79]: 9,
	_NodeState_name[79:86]: 10,
}

// NodeStateString returns the NodeState corresponding to the given string representation.
//
// The given string is lowercased before checking it against node states representations.
func NodeStateString(s string) (NodeState, error) {
	s = strings.ToLower(s)
	if val, ok := _NodeStateNameToValue_map[s]; ok {
		return val, nil
	}
	return 0, errors.Errorf("%s does not belong to State values", s)
}
