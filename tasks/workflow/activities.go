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

package workflow

import (
	"context"
	"sync"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/tasks/workflow/builder"
)

// An ActivityHook is a function that could be registered as pre or post activity hook and
// which is called respectively just before or after a workflow activity TaskExecution
type ActivityHook func(ctx context.Context, cfg config.Configuration, taskID, deploymentID, target string, activity builder.Activity)

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
