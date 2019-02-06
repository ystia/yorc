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

package terraform

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"io/ioutil"
	"path"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/helper/executil"
	"github.com/ystia/yorc/log"
	"github.com/ystia/yorc/prov"
	"github.com/ystia/yorc/prov/terraform/commons"
	"github.com/ystia/yorc/tasks"
	"github.com/ystia/yorc/tosca"
)

type defaultExecutor struct {
	generator       commons.Generator
	preDestroyCheck commons.PreDestroyInfraCallback
}

// NewExecutor returns an Executor
func NewExecutor(generator commons.Generator, preDestroyCheck commons.PreDestroyInfraCallback) prov.DelegateExecutor {
	return &defaultExecutor{generator: generator, preDestroyCheck: preDestroyCheck}
}

func (e *defaultExecutor) ExecDelegate(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	consulClient, err := cfg.GetConsulClient()
	if err != nil {
		return err
	}
	kv := consulClient.KV()

	instances, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}
	infrastructurePath := filepath.Join(cfg.WorkingDirectory, "deployments", deploymentID, "terraform", taskID, nodeName)
	if err = os.MkdirAll(infrastructurePath, 0775); err != nil {
		return errors.Wrapf(err, "Failed to create infrastructure working directory %q", infrastructurePath)
	}
	defer func() {
		if !cfg.Terraform.KeepGeneratedFiles {
			err := os.RemoveAll(infrastructurePath)
			if err != nil {
				err = errors.Wrapf(err, "Failed to remove Terraform infrastructure directory %q for node %q operation %q", infrastructurePath, nodeName, delegateOperation)
				log.Debugf("%+v", err)
				events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).RegisterAsString(err.Error())
			}
		}
	}()
	op := strings.ToLower(delegateOperation)
	switch {
	case op == "install":
		err = e.installNode(ctx, kv, cfg, deploymentID, nodeName, infrastructurePath, instances)
	case op == "uninstall":
		err = e.uninstallNode(ctx, kv, cfg, deploymentID, nodeName, infrastructurePath, instances)
	default:
		return errors.Errorf("Unsupported operation %q", delegateOperation)
	}
	return err
}

func (e *defaultExecutor) installNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, infrastructurePath string, instances []string) error {
	for _, instance := range instances {
		err := deployments.SetInstanceStateWithContextualLogs(events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: instance}), kv, deploymentID, nodeName, instance, tosca.NodeStateCreating)
		if err != nil {
			return err
		}
	}

	infraGenerated, outputs, env, cb, err := e.generator.GenerateTerraformInfraForNode(ctx, cfg, deploymentID, nodeName, infrastructurePath)
	if err != nil {
		return err
	}
	// Execute callback if needed
	defer func() {
		if cb != nil {
			cb()
		}
	}()
	if infraGenerated {
		if err = e.applyInfrastructure(ctx, kv, cfg, deploymentID, nodeName, infrastructurePath, outputs, env); err != nil {
			return err
		}
	}
	for _, instance := range instances {
		err := deployments.SetInstanceStateWithContextualLogs(events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: instance}), kv, deploymentID, nodeName, instance, tosca.NodeStateStarted)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *defaultExecutor) uninstallNode(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, infrastructurePath string, instances []string) error {
	for _, instance := range instances {
		err := deployments.SetInstanceStateWithContextualLogs(events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: instance}), kv, deploymentID, nodeName, instance, tosca.NodeStateDeleting)
		if err != nil {
			return err
		}
	}
	infraGenerated, outputs, env, cb, err := e.generator.GenerateTerraformInfraForNode(ctx, cfg, deploymentID, nodeName, infrastructurePath)
	if err != nil {
		return err
	}
	// Execute callback if needed
	defer func() {
		if cb != nil {
			cb()
		}
	}()
	if infraGenerated {
		if err = e.destroyInfrastructure(ctx, kv, cfg, deploymentID, nodeName, infrastructurePath, outputs, env); err != nil {
			return err
		}
	}
	for _, instance := range instances {
		err := deployments.SetInstanceStateWithContextualLogs(events.AddLogOptionalFields(ctx, events.LogOptionalFields{events.InstanceID: instance}), kv, deploymentID, nodeName, instance, tosca.NodeStateDeleted)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *defaultExecutor) remoteConfigInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, infrastructurePath string, env []string) error {
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString("Remote configuring the infrastructure")
	var cmd *executil.Cmd
	// Use pre-installed Terraform providers plugins if plugins directory exists
	// https://www.terraform.io/guides/running-terraform-in-automation.html#pre-installed-plugins
	if cfg.Terraform.PluginsDir != "" {
		cmd = executil.Command(ctx, "terraform", "init", "-input=false", "-plugin-dir="+cfg.Terraform.PluginsDir)
	} else {
		cmd = executil.Command(ctx, "terraform", "init")
	}

	cmd.Dir = infrastructurePath
	cmd.Env = mergeEnvironments(env)
	errbuf := events.NewBufferedLogEntryWriter()
	out := events.NewBufferedLogEntryWriter()
	cmd.Stdout = out
	cmd.Stderr = errbuf

	quit := make(chan bool)
	defer close(quit)

	// Register log entries via stderr/stdout buffers
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).RunBufferedRegistration(errbuf, quit)
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RunBufferedRegistration(out, quit)

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "Failed to setup Consul remote backend for terraform")
	}

	return errors.Wrap(cmd.Wait(), "Failed to setup Consul remote backend for terraform")
}

func (e *defaultExecutor) retrieveOutputs(ctx context.Context, kv *api.KV, infraPath string, outputs map[string]string) error {
	if len(outputs) == 0 {
		return nil
	}

	// Filter and handle file output
	filteredOutputs, err := e.handleFileOutputs(ctx, kv, infraPath, outputs)
	if err != nil {
		return err
	}

	if len(filteredOutputs) == 0 {
		return nil
	}

	type tfJSONOutput struct {
		Sensitive bool   `json:"sensitive,omitempty"`
		Type      string `json:"type,omitempty"`
		Value     string `json:"value,omitempty"`
	}
	type tfOutputsList map[string]tfJSONOutput

	cmd := executil.Command(ctx, "terraform", "output", "-json")
	cmd.Dir = infraPath
	result, err := cmd.Output()
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve the infrastructure outputs via terraform")
	}
	var outputsList tfOutputsList
	err = json.Unmarshal(result, &outputsList)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve the infrastructure outputs via terraform")
	}
	_, errGrp, store := consulutil.WithContext(ctx)
	for outPath, outName := range filteredOutputs {
		output, ok := outputsList[outName]
		if !ok {
			return errors.Errorf("failed to retrieve output %q in terraform result", outName)
		}
		store.StoreConsulKeyAsString(outPath, output.Value)
	}

	return errGrp.Wait()
}

// File outputs are outputs that terraform can't resolve and which need to be retrieved in local files
func (e *defaultExecutor) handleFileOutputs(ctx context.Context, kv *api.KV, infraPath string, outputs map[string]string) (map[string]string, error) {
	filteredOutputs := make(map[string]string, 0)
	for k, v := range outputs {
		if strings.HasPrefix(v, commons.FileOutputPrefix) {
			file := strings.TrimPrefix(v, commons.FileOutputPrefix)
			log.Debugf("Handle file output:%q", file)
			content, err := ioutil.ReadFile(path.Join(infraPath, file))
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to retrieve file output from file:%q", file)
			}
			contentStr := strings.Trim(string(content), "\r\n")
			// Store keyValue in Consul
			_, err = kv.Put(&api.KVPair{Key: k, Value: []byte(contentStr)}, nil)
			if err != nil {
				return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
			}
		} else {
			filteredOutputs[k] = v
		}
	}
	return filteredOutputs, nil
}

func (e *defaultExecutor) applyInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, infrastructurePath string, outputs map[string]string, env []string) error {

	// Remote Configuration for Terraform State to store it in the Consul KV store
	if err := e.remoteConfigInfrastructure(ctx, kv, cfg, deploymentID, nodeName, infrastructurePath, env); err != nil {
		return err
	}

	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString("Applying the infrastructure")
	cmd := executil.Command(ctx, "terraform", "apply", "-input=false", "-auto-approve")
	cmd.Dir = infrastructurePath
	cmd.Env = mergeEnvironments(env)
	errbuf := events.NewBufferedLogEntryWriter()
	out := events.NewBufferedLogEntryWriter()
	cmd.Stdout = out
	cmd.Stderr = errbuf

	quit := make(chan bool)
	defer close(quit)

	// Register log entries via stderr/stdout buffers
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).RunBufferedRegistration(errbuf, quit)
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RunBufferedRegistration(out, quit)

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "Failed to apply the infrastructure changes via terraform")
	}

	return e.retrieveOutputs(ctx, kv, infrastructurePath, outputs)

}

func (e *defaultExecutor) destroyInfrastructure(ctx context.Context, kv *api.KV, cfg config.Configuration, deploymentID, nodeName, infrastructurePath string, outputs map[string]string, env []string) error {
	if e.preDestroyCheck != nil {

		check, err := e.preDestroyCheck(ctx, kv, cfg, deploymentID, nodeName, infrastructurePath)
		if err != nil || !check {
			return err
		}
	}

	return e.applyInfrastructure(ctx, kv, cfg, deploymentID, nodeName, infrastructurePath, outputs, env)
}

// mergeEnvironments merges given env with current process env
// in commands if duplicates only the last one is taken into account
func mergeEnvironments(env []string) []string {
	return append(os.Environ(), env...)
}
