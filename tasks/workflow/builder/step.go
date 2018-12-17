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

package builder

// Step represents the workflow step
type Step struct {
	Name               string
	Target             string
	TargetRelationship string
	OperationHost      string
	Activities         []Activity
	Next               []*Step
	OnFailure          []*Step
	OnCancel           []*Step
	Previous           []*Step
	WorkflowName       string
	Async              bool
	IsOnFailurePath    bool
	IsOnCancelPath     bool
}

type visitStep struct {
	refCount int
	s        *Step
}

// IsInitial returns true is the workflow step has no previous step
func (s *Step) IsInitial() bool {
	return len(s.Previous) == 0
}

// IsTerminal returns true is the workflow step has no next step
func (s *Step) IsTerminal() bool {
	return len(s.Next) == 0
}
