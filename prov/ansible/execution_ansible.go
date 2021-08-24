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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/events"
)

const uploadArtifactsPlaybook = `
- name: Upload artifacts
  hosts: all
  strategy: free
  tasks:
[[[ range $artName, $art := .Artifacts ]]]
    [[[printf "- file: path=\"{{ ansible_env.HOME}}/%s/%s\" state=directory mode=0755" $.OperationRemotePath (path $art)]]]
    [[[printf "  when: ansible_os_family != 'Windows'"]]]
    [[[printf "- ansible.windows.win_file: path=\"{{ ansible_env.HOME}}\\%s\\%s\" state=directory mode=0755" $.OperationRemotePath (path $art)]]]
    [[[printf "  when: ansible_os_family == 'Windows'"]]]
    [[[printf "- copy: src=\"%s/%s\" dest=\"{{ ansible_env.HOME}}/%s/%s\"" $.OverlayPath $art $.OperationRemotePath (path $art)]]]
[[[end]]]
`

const ansiblePlaybook = `
- import_playbook: [[[.PlaybookPath]]]
[[[if .HaveOutput]]]
- name: Retrieving Operation outputs
  hosts: all
  strategy: free
  tasks:
    [[[printf "- file: path=\"{{ ansible_env.HOME}}/%s\" state=directory mode=0755" $.OperationRemotePath]]]
    [[[printf "  when: ansible_os_family != 'Windows'"]]]
    [[[printf "- ansible.windows.win_file: path=\"{{ ansible_env.HOME}}\\%s\" state=directory mode=0755" $.OperationRemotePath]]]
    [[[printf "  when: ansible_os_family == 'Windows'"]]]
    [[[printf "- template: src=\"outputs.csv.j2\" dest=\"{{ ansible_env.HOME}}/%s/out.csv\"" $.OperationRemotePath]]]
    [[[printf "- fetch: src=\"{{ ansible_env.HOME}}/%s/out.csv\" dest={{dest_folder}}/{{ansible_host}}-out.csv flat=yes" $.OperationRemotePath]]]
[[[end]]]
[[[if not .KeepOperationRemotePath]]]
- name: Cleanup temp directories
  hosts: all
  strategy: free
  tasks:
    - file: path="{{ ansible_env.HOME}}/[[[.OperationRemoteBaseDir]]]" state=absent
      when: ansible_os_family != 'Windows'
    - ansible.windows.win_file: path="{{ ansible_env.HOME}}\[[[.OperationRemoteBaseDir]]]" state=absent
      when: ansible_os_family == 'Windows'
[[[end]]]
`

type executionAnsible struct {
	*executionCommon
	PlaybookPath   string
	isAlienAnsible bool
}

func (e *executionAnsible) runAnsible(ctx context.Context, retry bool, currentInstance, ansibleRecipePath string) error {
	var err error
	if !e.isAlienAnsible {
		e.PlaybookPath, err = filepath.Abs(filepath.Join(e.OverlayPath, e.Primary))
	} else {
		var playbook string
		for _, envInput := range e.EnvInputs {
			if envInput.Name == "PLAYBOOK_ENTRY" {
				playbook = envInput.Value
				break
			}
		}
		if playbook == "" {
			err = errors.New("No PLAYBOOK_ENTRY input found for an alien4cloud ansible implementation")
		}
		if err == nil {
			e.PlaybookPath, err = filepath.Abs(filepath.Join(e.OverlayPath, filepath.Dir(e.Primary), playbook))
		}
	}
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}

	ansibleGroupsVarsPath := filepath.Join(ansibleRecipePath, "group_vars")
	if err = os.MkdirAll(ansibleGroupsVarsPath, 0775); err != nil {
		err = errors.Wrap(err, "Failed to create group_vars directory: ")
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}
	var buffer bytes.Buffer
	for _, envInput := range e.EnvInputs {
		if envInput.InstanceName != "" {
			buffer.WriteString(envInput.InstanceName)
			buffer.WriteString("_")
		}
		v, err := e.encodeEnvInputValue(envInput, ansibleRecipePath)
		if err != nil {
			return err
		}
		buffer.WriteString(fmt.Sprintf("%s: %s", envInput.Name, v))
		buffer.WriteString("\n")
	}

	if !e.isAlienAnsible {
		for artName, art := range e.Artifacts {
			buffer.WriteString(artName)
			buffer.WriteString(": \"{{ansible_env.HOME}}/")
			buffer.WriteString(e.OperationRemotePath)
			buffer.WriteString("/")
			buffer.WriteString(art)
			buffer.WriteString("\"\n")
		}
	} else {
		for artName, art := range e.Artifacts {
			buffer.WriteString(artName)
			buffer.WriteString(": \"")
			buffer.WriteString(e.OverlayPath)
			buffer.WriteString("/")
			buffer.WriteString(art)
			buffer.WriteString("\"\n")
		}
	}
	for contextKey, contextValue := range e.Context {
		buffer.WriteString(fmt.Sprintf("%s: %q", contextKey, contextValue))
		buffer.WriteString("\n")
	}
	for contextKey, contextValue := range e.CapabilitiesCtx {
		v, err := e.encodeTOSCAValue(contextValue, ansibleRecipePath)
		if err != nil {
			return err
		}
		buffer.WriteString(fmt.Sprintf("%s: %s", contextKey, v))
		buffer.WriteString("\n")
	}
	buffer.WriteString("dest_folder: \"")
	buffer.WriteString(ansibleRecipePath)
	buffer.WriteString("\"\n")

	if err = ioutil.WriteFile(filepath.Join(ansibleGroupsVarsPath, "all.yml"), buffer.Bytes(), 0664); err != nil {
		err = errors.Wrap(err, "Failed to write global group vars file: ")
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}

	if e.HaveOutput {
		buffer.Reset()
		for outputName := range e.Outputs {
			buffer.WriteString(outputName)
			buffer.WriteString(",{{")
			idx := strings.LastIndex(outputName, "_")
			buffer.WriteString(outputName[:idx])
			buffer.WriteString("}}\n")
		}
		if err = ioutil.WriteFile(filepath.Join(ansibleRecipePath, "outputs.csv.j2"), buffer.Bytes(), 0664); err != nil {
			err = errors.Wrap(err, "Failed to generate operation outputs file: ")
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
			return err
		}
	}

	buffer.Reset()
	funcMap := template.FuncMap{
		// The name "path" is what the function will be called in the template text.
		"path": filepath.Dir,
		"abs":  filepath.Abs,
		"cut":  cutAfterLastUnderscore,
	}
	tmpl := template.New("execTemplate").Delims("[[[", "]]]").Funcs(funcMap)
	var playbook string
	if e.isAlienAnsible || len(e.Artifacts) == 0 {
		playbook = ansiblePlaybook
	} else {
		playbook = uploadArtifactsPlaybook + ansiblePlaybook
	}
	tmpl, err = tmpl.Parse(playbook)
	if err != nil {
		err = errors.Wrap(err, "Failed to generate ansible playbook")
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}
	if err = tmpl.Execute(&buffer, e); err != nil {
		err = errors.Wrap(err, "Failed to Generate ansible playbook template")
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}
	if err = ioutil.WriteFile(filepath.Join(ansibleRecipePath, "run.ansible.yml"), buffer.Bytes(), 0664); err != nil {
		err = errors.Wrap(err, "Failed to write playbook file")
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, e.deploymentID).RegisterAsString(err.Error())
		return err
	}

	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelDEBUG, e.deploymentID).RegisterAsString(fmt.Sprintf("Ansible recipe for node %q: executing %q on remote host(s)", e.NodeName, filepath.Base(e.PlaybookPath)))

	outputHandler := &playbookOutputHandler{execution: e, context: ctx}
	return e.executePlaybook(ctx, retry, ansibleRecipePath, outputHandler)
}
