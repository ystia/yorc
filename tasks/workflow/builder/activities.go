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

//go:generate go-enum -f=activities.go --lower

// ActivityType x ENUM(
// delegate
// set-state
// call-operation
// inline
// )
type ActivityType int

// An Activity is the representation of a workflow activity
type Activity interface {
	// Type returns the ActivityType if this activity
	Type() ActivityType

	// Value returns the actual value of this activity
	//
	// For delegate activities it's the delegate operation name.
	// For set-state activities it's the state value.
	// For call-operation activities it's the operation name.
	// For inline activities it's the inlined workflow name.
	Value() string
}

type delegateActivity struct {
	delegate string
}

func (da delegateActivity) Type() ActivityType {
	return ActivityTypeDelegate
}
func (da delegateActivity) Value() string {
	return da.delegate
}

type setStateActivity struct {
	state string
}

func (s setStateActivity) Type() ActivityType {
	return ActivityTypeSetState
}
func (s setStateActivity) Value() string {
	return s.state
}

type callOperationActivity struct {
	operation string
}

func (c callOperationActivity) Type() ActivityType {
	return ActivityTypeCallOperation
}
func (c callOperationActivity) Value() string {
	return c.operation
}

type inlineActivity struct {
	inline string
}

func (i inlineActivity) Type() ActivityType {
	return ActivityTypeInline
}
func (i inlineActivity) Value() string {
	return i.inline
}
