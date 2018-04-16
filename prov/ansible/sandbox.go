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

package ansible

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/strslice"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/config"
)

func (e *executionCommon) createSandbox(ctx context.Context, sandboxCfg *config.DockerSandbox) (string, error) {

	// TODO make this global
	cli, err := client.NewEnvClient()
	if err != nil {
		return "", errors.Wrap(err, "Failed to connect to the docker daemon")
	}

	pullResp, err := cli.ImagePull(ctx, sandboxCfg.Image, types.ImagePullOptions{})
	if pullResp != nil {
		pullResp.Close()
	}
	if err != nil {
		return "", errors.Wrapf(err, "Failed to pull docker image %q", sandboxCfg.Image)
	}

	cc := &container.Config{
		Image: sandboxCfg.Image,
		Cmd:   strslice.StrSlice{sandboxCfg.Command},
	}

	hc := &container.HostConfig{
		AutoRemove: true,
	}

	createResp, err := cli.ContainerCreate(ctx, cc, hc, nil, "")
	if err != nil {
		return "", errors.Wrapf(err, "Failed to create docker sandbox %q", sandboxCfg.Image)
	}

	err = cli.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
	if err != nil {
		timeout := 10 * time.Second
		cli.ContainerStop(ctx, createResp.ID, &timeout)
		return "", errors.Wrapf(err, "Failed to create docker sandbox %q", sandboxCfg.Image)
	}

	return createResp.ID, nil
}
