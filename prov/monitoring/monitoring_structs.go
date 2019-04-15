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

package monitoring

import (
	"context"
	"sync"
	"time"
)

//go:generate go-enum -f=monitoring_structs.go --lower

// CheckStatus x ENUM(
// INITIAL,
// PASSING,
// CRITICAL
// WARNING
// )
type CheckStatus int

// CheckType x ENUM(
// TCP,
// HTTP
// )
type CheckType int

// Check represents a registered check
type Check struct {
	ID           string
	TimeInterval time.Duration
	Report       CheckReport
	CheckType    CheckType

	stop      bool
	stopLock  sync.Mutex
	chStop    chan struct{}
	timeout   time.Duration
	ctx       context.Context
	execution checkExecution
}

// CheckReport represents a node check report including its status
type CheckReport struct {
	DeploymentID string
	NodeName     string
	Instance     string
	Status       CheckStatus
}
