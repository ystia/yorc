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
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/opts"
	"io/ioutil"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/api/types/versions"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/log"
)

func createSandbox(ctx context.Context, cli *client.Client, sandboxCfg *config.DockerSandbox, deploymentID,
	ansibleRecipePath, overlayPath, sshAgentSocket string, env []string) (string, error) {

	// check context is cancelable
	if ctx.Done() == nil {
		return "", errors.New("should provide a cancelable context for creating a docker sandbox")
	}

	// At least sandboxCfg.Image is required
	if sandboxCfg.Image == "" {
		return "", errors.New("Docker sandbox for orchestrator-hosted operation misconfigured, image option is missing")
	}
	// pull docker image if not exists
	_, _, err := cli.ImageInspectWithRaw(ctx, sandboxCfg.Image)
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("Docker image %s not exists. "+
			"Pulling docker image.", sandboxCfg.Image)
		pullResp, err := cli.ImagePull(ctx, sandboxCfg.Image, types.ImagePullOptions{})
		if pullResp != nil {
			b, errRead := ioutil.ReadAll(pullResp)
			if errRead == nil && len(b) > 0 {
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("Pulled docker image: %s", string(b))
			}
			pullResp.Close()
		}
		if err != nil {
			return "", errors.Wrapf(err, "Failed to pull docker image %q", sandboxCfg.Image)
		}
	}
	// set the home environment for the sandbox container so that ansible resolves its default path (~/.ansible)
	// to the work dir inside the container
	env = append(env, "HOME="+config.DefaultSandboxWorkDir)
	env = append(env, sandboxCfg.Env...)
	cc := &container.Config{
		Image: sandboxCfg.Image,
		Env:   env,
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
	// run sandbox container with non-root user
	if sandboxCfg.User != "" {
		cc.User = sandboxCfg.User
	}
	// limit resources for sandbox container to avoid DoS attacks
	c, err := getNanoCPUs(sandboxCfg.Cpus)
	if err != nil {
		return "", err
	}
	m, err := getMemoryInBytes(sandboxCfg.Memory)
	if err != nil {
		return "", err
	}

	hc := &container.HostConfig{
		AutoRemove: true,
		// Security hardening for the sandbox container
		// do not allow privilege escalation
		SecurityOpt: []string{"no-new-privileges=true"},
		// run the sandbox container with read-only root file system
		ReadonlyRootfs: true,
		// drop all capabilities
		CapDrop: []string{"ALL"},
		Resources: container.Resources{
			NanoCPUs: c,
			Memory:   m,
		},
		// mount volumes
		Mounts: []mount.Mount{
			{
				Type:     "bind",
				Source:   ansibleRecipePath,
				Target:   config.DefaultSandboxWorkDir,
				ReadOnly: false,
			},
			{
				Type:     "bind",
				Source:   overlayPath,
				Target:   config.DefaultSandboxOverlayDir,
				ReadOnly: true,
			},
		},
	}

	if sshAgentSocket != "" {
		hc.Mounts = append(hc.Mounts, mount.Mount{
			Type:     "bind",
			Source:   sshAgentSocket,
			Target:   config.DefaultSandboxMountAgentSocket,
			ReadOnly: true,
		})
	}

	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("Creating docker container from image: %s", sandboxCfg.Image)
	createResp, err := cli.ContainerCreate(ctx, cc, hc, nil, "")
	if err != nil {
		return "", errors.Wrapf(err, "Failed to create docker sandbox %q", sandboxCfg.Image)
	}

	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("Docker container with id %q created", createResp.ID)
	err = cli.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
	if err != nil {
		timeout := 10 * time.Second
		cli.ContainerStop(ctx, createResp.ID, &timeout)
		return "", errors.Wrapf(err, "Failed to create docker sandbox %q", sandboxCfg.Image)
	}
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("Docker container with id %q started", createResp.ID)
	go stopSandboxOnContextCancellation(ctx, cli, deploymentID, createResp.ID)
	return createResp.ID, nil
}

func stopSandboxOnContextCancellation(ctx context.Context, cli *client.Client, deploymentID, containerID string) {
	<-ctx.Done()
	timeout := 10 * time.Second
	err := cli.ContainerStop(context.Background(), containerID, &timeout)
	if err != nil {
		log.Printf("Failed to delete docker container %v", err)
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelWARN, deploymentID).Registerf("Failed to delete your docker container execution sandbox %q. Please retport this to your system administrator.", containerID)
	}
	if versions.LessThan(cli.ClientVersion(), "1.25") {
		// auto-remove is disable before 1.25
		cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
	}
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, deploymentID).Registerf("Docker container with id %q removed", containerID)
}

// getNanoCPUs converts user defined cpus string to nano cpus presentation
// defaults cpu to 0.5 cpu if not set
func getNanoCPUs(c string) (int64, error) {
	var nanoCPUs opts.NanoCPUs
	if c != "" {
		if err := nanoCPUs.Set(c); err != nil {
			return 0, err
		}
	} else {
		_ = nanoCPUs.Set("0.5")
	}
	return nanoCPUs.Value(), nil
}

// getMemoryInBytes converts user defined memory string to memory in bytes presentation
// defaults memory to 256m if not set
func getMemoryInBytes(m string) (int64, error) {
	var memoryInBytes opts.MemBytes
	if m != "" {
		if err := memoryInBytes.Set(m); err != nil {
			return 0, err
		}
	} else {
		_ = memoryInBytes.Set("256m")
	}
	return memoryInBytes.Value(), nil
}
