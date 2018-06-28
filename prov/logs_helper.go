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

package prov

import (
	"context"

	"github.com/ystia/yorc/events"
)

// AddInstanceToContextLogFields adds the given instance name to the list of existing events.LogOptionalFields
// attached to the context (creating them if missing).
//
// Any existing instance id is overwriten.
func AddInstanceToContextLogFields(ctx context.Context, instanceName string) context.Context {
	lof, ok := events.FromContext(ctx)
	if !ok {
		lof = make(events.LogOptionalFields)
	}
	lof[events.InstanceID] = instanceName
	return events.NewContext(ctx, lof)
}
