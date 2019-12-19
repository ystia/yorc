// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package rest

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/ystia/yorc/v4/deployments"
)

func checkBlockingOperationOnDeployment(ctx context.Context, deploymentID string, w http.ResponseWriter, r *http.Request) bool {
	hasBlocking, err := deployments.HasBlockingOperationOnDeploymentFlag(ctx, deploymentID)
	if err != nil {
		log.Panicf("%v", err)
	}
	if hasBlocking {
		writeError(w, r, newBadRequestError(fmt.Errorf("deployment %q, is currently processing a blocking operation we can't proceed with your request until this operation finish", deploymentID)))
		return false
	}
	return true
}
