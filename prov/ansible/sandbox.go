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
	"io/ioutil"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/api/types/versions"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/log"
)

func createSandbox(ctx context.Context, cli *client.Client, sandboxCfg *config.DockerSandbox, deploymentID string) (string, error) {

	// check context is cancelable
	if ctx.Done() == nil {
		return "", errors.New("should provide a cancelable context for creating a docker sandbox")
	}

	pullResp, err := cli.ImagePull(ctx, sandboxCfg.Image, types.ImagePullOptions{})
	if pullResp != nil {
		b, errRead := ioutil.ReadAll(pullResp)
		if errRead == nil && len(b) > 0 {
			log.Debugf("Pulled docker image image: %s", string(b))
			events.WithContextOptionalFields(ctx).NewLogEntry(events.DEBUG, deploymentID).Registerf("Pulled docker image image: %s", string(b))
		}
		pullResp.Close()
	}
	if err != nil {
		return "", errors.Wrapf(err, "Failed to pull docker image %q", sandboxCfg.Image)
	}

	cc := &container.Config{
		Image: sandboxCfg.Image,
	}

	if len(sandboxCfg.Command) == 0 && len(sandboxCfg.Entrypoint) == 0 {
		cc.Entrypoint = strslice.StrSlice{"python"}
		cc.Cmd = strslice.StrSlice{"-c", "import time;time.sleep(31536000);"}
	}
	if len(sandboxCfg.Command) > 0 {
		cc.Cmd = strslice.StrSlice(sandboxCfg.Command)
	}
	if len(sandboxCfg.Entrypoint) > 0 {
		cc.Entrypoint = strslice.StrSlice(sandboxCfg.Entrypoint)
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
	go stopSandboxOnContextCancellation(ctx, cli, deploymentID, createResp.ID)
	return createResp.ID, nil
}

func stopSandboxOnContextCancellation(ctx context.Context, cli *client.Client, deploymentID, containerID string) {
	<-ctx.Done()
	timeout := 10 * time.Second
	err := cli.ContainerStop(context.Background(), containerID, &timeout)
	if err != nil {
		log.Printf("Failed to delete docker container %v", err)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).Registerf("Failed to delete your docker container execution sandbox %q. Please retport this to your system administrator.", containerID)
	}
	if versions.LessThan(cli.ClientVersion(), "1.25") {
		// auto-remove is disable before 1.25
		cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
	}
}
