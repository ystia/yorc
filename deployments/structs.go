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

package deployments

//go:generate stringer -type=DeploymentStatus structs.go

// DeploymentStatus represent the current status of a deployment
type DeploymentStatus int

const (
	startOfDepStatusConst DeploymentStatus = iota // Do not remove this line and define new const after it. It is used to get const value from string
	// INITIAL is the initial status of a deployment when it has not yet started
	INITIAL
	// DEPLOYMENT_IN_PROGRESS deployment is in progress
	DEPLOYMENT_IN_PROGRESS
	// DEPLOYED deployment is deployed without error
	DEPLOYED
	// UNDEPLOYMENT_IN_PROGRESS undeployment is in progress
	UNDEPLOYMENT_IN_PROGRESS
	// UNDEPLOYED deployment is no longer deployed
	UNDEPLOYED
	// DEPLOYMENT_FAILED deployment encountered an error
	DEPLOYMENT_FAILED
	// UNDEPLOYMENT_FAILED undeployment encountered an error
	UNDEPLOYMENT_FAILED
	// SCALING_IN_PROGRESS instances are currently added or removed to the deployment
	SCALING_IN_PROGRESS

	endOfDepStatusConst // Do not remove this line and define new const before it. It is used to get const value from string
)
