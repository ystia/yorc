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

package commons

import (
	"context"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/helper/sshutil"
)

// The contextKey type is unexported to prevent collisions with context keys defined in
// other packages.
type contextKey int

// sshAgentkey is the context key for the ssh agent.  Its value of zero is
// arbitrary.  If this package defined other context keys, they would have
// different integer values.
const sshAgentkey contextKey = 0

// StoreSSHAgentInContext allows to store an sshutil.SSHAgent into the given Context
func StoreSSHAgentInContext(ctx context.Context, agent *sshutil.SSHAgent) context.Context {
	return context.WithValue(ctx, sshAgentkey, agent)
}

// SSHAgentFromContext retrieves an sshutil.SSHAgent from the given Context
func SSHAgentFromContext(ctx context.Context) (*sshutil.SSHAgent, bool) {
	existingAgent, ok := ctx.Value(sshAgentkey).(*sshutil.SSHAgent)
	return existingAgent, ok
}

func addKeysToContextualSSHAgent(ctx context.Context, keys []*sshutil.PrivateKey) {
	addKeysToContextualSSHAgentOrFail(ctx, keys)
}

func addKeysToContextualSSHAgentOrFail(ctx context.Context, keys []*sshutil.PrivateKey) error {
	agent, ok := SSHAgentFromContext(ctx)
	if !ok {
		return errors.New("no contextual ssh-agent")
	}
	for _, key := range keys {
		agent.AddPrivateKey(key, 3600)
	}
	return nil
}
