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

package dispatcher

import (
	"context"
	"sync"

	"github.com/ystia/yorc/config"
)

//go:generate go-enum -f=activities.go --lower

// An ActivityHook is a function that could be registered as pre or post activity hook and
// which is called respectively just before or after a workflow activity TaskExecution
type ActivityHook func(ctx context.Context, cfg config.Configuration, taskID, deploymentID, target string, activity Activity)

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

// RegisterPreActivityHook registers an ActivityHook in the list of ActivityHooks that will
// be triggered before a workflow activity
func RegisterPreActivityHook(activityHook ActivityHook) {
	activityHookslock.Lock()
	defer activityHookslock.Unlock()
	preActivityHooks = append(preActivityHooks, activityHook)
}

// RegisterPostActivityHook registers an ActivityHook in the list of ActivityHooks that will
// be triggered after a workflow activity
func RegisterPostActivityHook(activityHook ActivityHook) {
	activityHookslock.Lock()
	defer activityHookslock.Unlock()
	postActivityHooks = append(postActivityHooks, activityHook)
}

var activityHookslock sync.Mutex
var preActivityHooks = make([]ActivityHook, 0)
var postActivityHooks = make([]ActivityHook, 0)

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
